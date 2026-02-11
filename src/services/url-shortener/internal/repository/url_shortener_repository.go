package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"url-shortener/internal/cache"
	"url-shortener/internal/config"
	"url-shortener/internal/models"
	"url-shortener/internal/utils"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type URLShortenerRepository struct {
	db    *gorm.DB
	cfg   *config.Config
	cache cache.Cache
}

func NewURLShortenerRepository(db *gorm.DB, cfg *config.Config, c cache.Cache) *URLShortenerRepository {
	return &URLShortenerRepository{
		db:    db,
		cfg:   cfg,
		cache: c,
	}
}

func (r *URLShortenerRepository) GetOrCreateShortKey(req models.ShortenRequest) (string, error) {
	keyLen := r.cfg.ShortKeyMinLength
	maxAttempts := r.cfg.MaxShortKeyRetries

	// 1) Try to find if URL already exists
	var existing models.URLShortener
	tx := r.db.Where("url = ?", req.URL).First(&existing)
	if tx.Error == nil {
		return existing.ShortKey, nil
	}
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return "", tx.Error
	}

	// 2) Create new entry with a generated short key (handle collisions)
	for i := 0; i < maxAttempts; i++ {
		k, err := utils.GenerateShortKey(keyLen)
		if err != nil {
			return "", err
		}

		newRow := models.URLShortener{
			ShortKey:  k,
			URL:       req.URL,
			ValidFrom: req.ValidFrom,
			ValidTill: req.ValidTill,
		}
		if err := r.db.Create(&newRow).Error; err != nil {
			// Inspect DB error to decide whether to retry or fail.
			// If it's duplicate URL, return existing
			// If it's duplicate shortkey, retry
			// Otherwise, return the error.
			if isUniqueViolation(err) {
				// try find by URL (maybe race)
				var maybe models.URLShortener
				if err2 := r.db.Where("url = ?", req.URL).First(&maybe).Error; err2 == nil {
					return maybe.ShortKey, nil
				}
				// If URL not found, likely shortkey collision -> retry
				continue
			}
			// unexpected DB error
			return "", err
		}
		// success: set cache (best-effort)
		if cerr := r.cache.Set(context.Background(), cacheKeyForShortKey(k), req.URL, r.cfg.CacheTTL); cerr != nil {
			log.Printf("warning: failed to set cache for key=%s: %v", k, cerr)
		}
		return k, nil
	}

	return "", fmt.Errorf("failed to generate unique short key after %d attempts", maxAttempts)
}

func (r *URLShortenerRepository) GetURL(shortKey string) (string, error) {
	ctx := context.Background()
	if v, err := r.cache.Get(ctx, cacheKeyForShortKey(shortKey)); err == nil {
		return v, nil
	}

	var row models.URLShortener
	if err := r.db.Where("shortkey = ?", shortKey).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("key not found")
		}
		return "", err
	}

	now := time.Now().Unix()
	if row.ValidFrom != 0 && now < row.ValidFrom {
		return "", fmt.Errorf("not active yet")
	}
	if row.ValidTill != 0 && now > row.ValidTill {
		return "", fmt.Errorf("expired")
	}

	_ = r.cache.Set(ctx, cacheKeyForShortKey(shortKey), row.URL, r.cfg.CacheTTL)
	return row.URL, nil
}

func cacheKeyForShortKey(k string) string {
	return "url_short:" + k
}

func isUniqueViolation(err error) bool {
	// Try pgconn.PgError (pgx)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// code 23505 is unique_violation in Postgres
		return pgErr.Code == "23505"
	}
	// Try lib/pq
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == "23505"
	}
	// Fallback: look for unique/duplicate in the message
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
		return true
	}
	return false
}
