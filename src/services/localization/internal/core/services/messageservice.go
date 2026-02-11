package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"localization/internal/core/domain"
	"localization/internal/core/ports"
)

// MessageServiceImpl is the implementation of MessageService
type MessageServiceImpl struct {
	repository        ports.MessageRepository
	cache             ports.MessageCache
	messageLocalesMap map[string]map[string]map[string][]string // tenantID -> module -> code -> []locale
	mapMutex          sync.RWMutex                              // Protects messageLocalesMap from concurrent access
}

// NewMessageService creates a new message service
func NewMessageService(repository ports.MessageRepository, cache ports.MessageCache) ports.MessageService {
	return &MessageServiceImpl{
		repository:        repository,
		cache:             cache,
		messageLocalesMap: make(map[string]map[string]map[string][]string),
	}
}

// LoadAllMessages loads all messages from the repository and builds the tenant-to-code-to-locales map
func (s *MessageServiceImpl) LoadAllMessages(ctx context.Context) error {
	log.Println("Loading all messages into cache...")
	messages, err := s.repository.FindAllMessages(ctx)
	if err != nil {
		return err
	}

	// Lock the map for writing during rebuild
	s.mapMutex.Lock()
	defer s.mapMutex.Unlock()

	// Clear the existing map
	s.messageLocalesMap = make(map[string]map[string]map[string][]string)

	for _, msg := range messages {
		if _, ok := s.messageLocalesMap[msg.TenantID]; !ok {
			s.messageLocalesMap[msg.TenantID] = make(map[string]map[string][]string)
		}
		if _, ok := s.messageLocalesMap[msg.TenantID][msg.Module]; !ok {
			s.messageLocalesMap[msg.TenantID][msg.Module] = make(map[string][]string)
		}
		s.messageLocalesMap[msg.TenantID][msg.Module][msg.Code] = append(s.messageLocalesMap[msg.TenantID][msg.Module][msg.Code], msg.Locale)
	}
	log.Println("Finished loading all messages into cache.")
	return nil
}

// FindMissingMessages finds the missing messages for a given tenant.
// The response can be filtered by providing a module.
func (s *MessageServiceImpl) FindMissingMessages(ctx context.Context, tenantID string, module string) (map[string]map[string][]string, error) {
	// Lock for reading the map
	s.mapMutex.RLock()
	defer s.mapMutex.RUnlock()

	tenantData, ok := s.messageLocalesMap[tenantID]
	if !ok {
		return nil, nil // Or an error indicating tenant not found
	}

	modulesToProcess := make(map[string]map[string][]string)
	if module != "" {
		if moduleData, ok := tenantData[module]; ok {
			modulesToProcess[module] = moduleData
		}
	} else {
		modulesToProcess = tenantData
	}

	result := make(map[string]map[string][]string)

	for moduleName, moduleData := range modulesToProcess {
		// Determine the set of all unique codes and locales for the module.
		allCodesForModule := make(map[string]struct{})
		allLocalesForModule := make(map[string]struct{})
		for code, locales := range moduleData {
			allCodesForModule[code] = struct{}{}
			for _, l := range locales {
				allLocalesForModule[l] = struct{}{}
			}
		}

		missingMessagesForModule := make(map[string][]string)
		for locale := range allLocalesForModule {
			var missingCodes []string
			for code := range allCodesForModule {
				// Check if the current code exists for the current locale in this module
				found := false
				if localesForCode, ok := moduleData[code]; ok {
					for _, l := range localesForCode {
						if l == locale {
							found = true
							break
						}
					}
				}
				if !found {
					missingCodes = append(missingCodes, code)
				}
			}
			if len(missingCodes) > 0 {
				missingMessagesForModule[locale] = missingCodes
			}
		}
		if len(missingMessagesForModule) > 0 {
			result[moduleName] = missingMessagesForModule
		}
	}

	return result, nil
}

// UpsertMessages creates or updates localization messages
func (s *MessageServiceImpl) UpsertMessages(ctx context.Context, tenantID string, userID string, messages []domain.Message) ([]domain.Message, error) {
	messagesToCreate := []domain.Message{}
	messagesToUpdate := []domain.Message{}

	for _, msg := range messages {
		if msg.UUID == "" {
			messagesToCreate = append(messagesToCreate, msg)
		} else {
			messagesToUpdate = append(messagesToUpdate, msg)
		}
	}

	now := time.Now()
	var finalMessages []domain.Message

	// Handle creations
	if len(messagesToCreate) > 0 {
		for i := range messagesToCreate {
			messagesToCreate[i].UUID = uuid.New().String()
			messagesToCreate[i].TenantID = tenantID
			messagesToCreate[i].CreatedDate = now
			messagesToCreate[i].LastModifiedDate = now
			messagesToCreate[i].CreatedBy = userID
			messagesToCreate[i].LastModifiedBy = userID
		}
		err := s.repository.SaveMessages(ctx, messagesToCreate)
		if err != nil {
			return nil, err
		}
		finalMessages = append(finalMessages, messagesToCreate...)

		// Update the in-memory map for created messages
		s.mapMutex.Lock()
		for _, msg := range messagesToCreate {
			if _, ok := s.messageLocalesMap[msg.TenantID]; !ok {
				s.messageLocalesMap[msg.TenantID] = make(map[string]map[string][]string)
			}
			if _, ok := s.messageLocalesMap[msg.TenantID][msg.Module]; !ok {
				s.messageLocalesMap[msg.TenantID][msg.Module] = make(map[string][]string)
			}

			// Avoid adding duplicate locales for a code
			found := false
			if locales, ok := s.messageLocalesMap[msg.TenantID][msg.Module][msg.Code]; ok {
				for _, locale := range locales {
					if locale == msg.Locale {
						found = true
						break
					}
				}
			}
			if !found {
				s.messageLocalesMap[msg.TenantID][msg.Module][msg.Code] = append(s.messageLocalesMap[msg.TenantID][msg.Module][msg.Code], msg.Locale)
			}
		}
		s.mapMutex.Unlock()
	}

	// Handle updates
	if len(messagesToUpdate) > 0 {
		for i := range messagesToUpdate {
			messagesToUpdate[i].TenantID = tenantID
			messagesToUpdate[i].LastModifiedDate = now
			messagesToUpdate[i].LastModifiedBy = userID
		}
		updatedMessages, err := s.repository.UpdateMessages(ctx, tenantID, messagesToUpdate)
		if err != nil {
			return nil, err
		}
		finalMessages = append(finalMessages, updatedMessages...)
	}

	// Invalidate cache
	s.invalidateCacheForMessages(ctx, finalMessages)

	return finalMessages, nil
}

// CreateMessages creates new localization messages
func (s *MessageServiceImpl) CreateMessages(ctx context.Context, tenantID string, userID string, messages []domain.Message) ([]domain.Message, error) {
	// Enrich messages with tenant ID and timestamps
	now := time.Now()
	for i := range messages {
		messages[i].UUID = uuid.New().String()
		messages[i].TenantID = tenantID
		messages[i].CreatedDate = now
		messages[i].LastModifiedDate = now
		messages[i].CreatedBy = userID
		messages[i].LastModifiedBy = userID
	}

	// Save to database
	err := s.repository.SaveMessages(ctx, messages)
	if err != nil {
		return nil, err
	}

	// Invalidate cache for affected tenant+module+locale combinations
	s.invalidateCacheForMessages(ctx, messages)

	// Update the in-memory map
	s.mapMutex.Lock()
	for _, msg := range messages {
		if _, ok := s.messageLocalesMap[msg.TenantID]; !ok {
			s.messageLocalesMap[msg.TenantID] = make(map[string]map[string][]string)
		}
		if _, ok := s.messageLocalesMap[msg.TenantID][msg.Module]; !ok {
			s.messageLocalesMap[msg.TenantID][msg.Module] = make(map[string][]string)
		}
		// Avoid adding duplicate locales for a code
		found := false
		if locales, ok := s.messageLocalesMap[msg.TenantID][msg.Module][msg.Code]; ok {
			for _, locale := range locales {
				if locale == msg.Locale {
					found = true
					break
				}
			}
		}
		if !found {
			s.messageLocalesMap[msg.TenantID][msg.Module][msg.Code] = append(s.messageLocalesMap[msg.TenantID][msg.Module][msg.Code], msg.Locale)
		}
	}
	s.mapMutex.Unlock()

	return messages, nil
}

// UpdateMessages updates existing messages by their UUIDs
func (s *MessageServiceImpl) UpdateMessages(ctx context.Context, tenantID string, userID string, messages []domain.Message) ([]domain.Message, error) {
	// Enrich messages with timestamp
	now := time.Now()
	for i := range messages {
		messages[i].TenantID = tenantID
		messages[i].LastModifiedDate = now
		messages[i].LastModifiedBy = userID
	}

	// Update in database
	updatedMessages, err := s.repository.UpdateMessages(ctx, tenantID, messages)
	if err != nil {
		return nil, err
	}

	// Invalidate cache for affected tenant+module+locale combinations
	s.invalidateCacheForMessages(ctx, updatedMessages)

	return updatedMessages, nil
}

// DeleteMessages deletes messages by their UUIDs
func (s *MessageServiceImpl) DeleteMessages(ctx context.Context, tenantID string, uuids []string) error {
	// Note: To invalidate the cache correctly, we would first need to fetch the messages
	// to get their module and locale. For simplicity in this example, we will
	// bust the entire cache for the tenant. A more optimized approach would fetch
	// the details before deleting.
	if err := s.cache.BustCache(ctx, tenantID, "", ""); err != nil {
		log.Printf("Failed to bust cache for tenant %s during delete, continuing...", tenantID)
	}

	return s.repository.DeleteMessages(ctx, tenantID, uuids)
}

// BustCache clears the cache based on tenant, and optionally module and locale
func (s *MessageServiceImpl) BustCache(ctx context.Context, tenantID, module, locale string) error {
	cacheKey := fmt.Sprintf("tenant:%s|module:%s|locale:%s", tenantID, module, locale)
	start := time.Now()

	err := s.cache.BustCache(ctx, tenantID, module, locale)
	duration := time.Since(start)

	if err != nil {
		log.Printf("CACHE: BUST ERROR - %s (error: %v, duration: %v)", cacheKey, err, duration)
		return err
	}

	log.Printf("CACHE: BUST - %s (duration: %v)", cacheKey, duration)
	return nil
}

// SearchMessages searches for messages, checking the cache first
func (s *MessageServiceImpl) SearchMessages(ctx context.Context, tenantID, module, locale string, limit, offset int) ([]domain.Message, error) {
	cacheKey := fmt.Sprintf("tenant:%s|module:%s|locale:%s", tenantID, module, locale)

	// Bypass cache for paginated requests. Caching paginated results is complex.
	if limit > 0 {
		log.Printf("CACHE: BYPASS for paginated request - %s (limit:%d, offset:%d)", cacheKey, limit, offset)
		return s.repository.FindMessages(ctx, tenantID, module, locale, limit, offset)
	}

	// Try to get from cache first for non-paginated requests
	start := time.Now()
	if cachedMessages, err := s.cache.GetMessages(ctx, tenantID, module, locale); err == nil && len(cachedMessages) > 0 {
		cacheDuration := time.Since(start)
		log.Printf("CACHE: HIT - %s (found %d messages in %v)", cacheKey, len(cachedMessages), cacheDuration)
		return cachedMessages, nil
	} else {
		cacheDuration := time.Since(start)
		if err != nil && err != ports.ErrCacheMiss {
			log.Printf("CACHE: ERROR - %s (error: %v, duration: %v)", cacheKey, err, cacheDuration)
		} else {
			log.Printf("CACHE: MISS - %s (duration: %v)", cacheKey, cacheDuration)
		}
	}

	// If not in cache, get from repository (non-paginated)
	dbStart := time.Now()
	messages, err := s.repository.FindMessages(ctx, tenantID, module, locale, 0, 0) // limit=0 means no limit
	if err != nil {
		dbDuration := time.Since(dbStart)
		log.Printf("DB: ERROR - %s (error: %v, duration: %v)", cacheKey, err, dbDuration)
		return nil, err
	}
	dbDuration := time.Since(dbStart)
	log.Printf("DB: FETCH - %s (found %d messages in %v)", cacheKey, len(messages), dbDuration)

	// Store in cache
	cacheSetStart := time.Now()
	if err := s.cache.SetMessages(ctx, tenantID, module, locale, messages); err != nil {
		cacheSetDuration := time.Since(cacheSetStart)
		log.Printf("CACHE: SET ERROR - %s (error: %v, duration: %v)", cacheKey, err, cacheSetDuration)
	} else {
		cacheSetDuration := time.Since(cacheSetStart)
		log.Printf("CACHE: SET - %s (stored %d messages in %v)", cacheKey, len(messages), cacheSetDuration)
	}

	return messages, nil
}

// SearchMessagesByCodes searches for messages by specific codes, checking the cache first
func (s *MessageServiceImpl) SearchMessagesByCodes(ctx context.Context, tenantID, locale string, codes []string) ([]domain.Message, error) {
	// For searching by codes, it is complex to implement a partial cache read.
	// For now, we will bypass the cache and go straight to the repository.
	// A more advanced caching strategy could be implemented here in the future.
	return s.repository.FindMessagesByCode(ctx, tenantID, locale, codes)
}

// invalidateCacheForMessages invalidates the cache for the given messages
func (s *MessageServiceImpl) invalidateCacheForMessages(ctx context.Context, messages []domain.Message) {
	cacheKeys := make(map[string]struct{})
	for _, msg := range messages {
		key := msg.TenantID + ":" + msg.Module + ":" + msg.Locale
		cacheKeys[key] = struct{}{}
	}

	for key := range cacheKeys {
		parts := strings.Split(key, ":")
		if len(parts) == 3 {
			start := time.Now()
			if err := s.cache.Invalidate(ctx, parts[0], parts[1], parts[2]); err != nil {
				duration := time.Since(start)
				log.Printf("CACHE: INVALIDATE ERROR - tenant:%s|module:%s|locale:%s (error: %v, duration: %v)",
					parts[0], parts[1], parts[2], err, duration)
			} else {
				duration := time.Since(start)
				log.Printf("CACHE: INVALIDATE - tenant:%s|module:%s|locale:%s (duration: %v)",
					parts[0], parts[1], parts[2], duration)
			}
		}
	}
}
