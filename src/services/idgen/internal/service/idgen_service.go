package service

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"idgen/internal/models"
	"idgen/internal/repository"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IDGenService struct {
	repo *repository.IDGenRepository
}

func NewIDGenService(repo *repository.IDGenRepository) *IDGenService {
	return &IDGenService{repo: repo}
}

// Keyword-to-Go-date layout map
var dateKeywordMap = map[string]string{
	// Basic numeric formats
	"yyyymmdd": "20060102", // 20250922
	"ddmmyyyy": "02012006", // 22092025
	"mmddyyyy": "01022006", // 09222025
	"yymmdd":   "060102",   // 250922
	"ddmmyy":   "020106",   // 220925
	"mmddyy":   "010206",   // 092205

	// Dash-separated
	"yyyy-mm-dd": "2006-01-02", // 2025-09-22
	"dd-mm-yyyy": "02-01-2006", // 22-09-2025
	"mm-dd-yyyy": "01-02-2006", // 09-22-2025
	"yy-mm-dd":   "06-01-02",   // 25-09-22
	"dd-mm-yy":   "02-01-06",   // 22-09-25

	// Slash-separated
	"yyyy/mm/dd": "2006/01/02", // 2025/09/22
	"dd/mm/yyyy": "02/01/2006", // 22/09/2025
	"mm/dd/yyyy": "01/02/2006", // 09/22/2025
	"yy/mm/dd":   "06/01/02",   // 25/09/22
	"dd/mm/yy":   "02/01/06",   // 22/09/25

	// Dot-separated
	"yyyy.mm.dd": "2006.01.02", // 2025.09.22
	"dd.mm.yyyy": "02.01.2006", // 22.09.2025
	"mm.dd.yyyy": "01.02.2006", // 09.22.2025
	"yy.mm.dd":   "06.01.02",   // 25.09.22
	"dd.mm.yy":   "02.01.06",   // 22.09.25

	// Month-Year combinations
	"mmyyyy":  "012006",  // 092025
	"mm-yyyy": "01-2006", // 09-2025
	"mm/yyyy": "01/2006", // 09/2025
	"mm.yyyy": "01.2006", // 09.2025
	"yyyy-mm": "2006-01", // 2025-09
	"yyyy/mm": "2006/01", // 2025/09
	"yyyy.mm": "2006.01", // 2025.09

	// Year only
	"yyyy": "2006", // 2025
	"yy":   "06",   // 25
}

func parseVersion(versionStr string) int {
	if versionStr == "" {
		return 0
	}
	// Remove leading "v" or "V"
	versionStr = strings.TrimPrefix(strings.ToLower(versionStr), "v")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		log.Printf("[WARN] Invalid version format: %s", versionStr)
		return 0
	}
	return version
}

// CreateTemplate creates the first version of a template (version v1)
func (s *IDGenService) CreateTemplate(req *models.IDGenTemplate) (*models.IDGenTemplateDB, error) {
	// Check if template already exists
	existing, err := s.repo.GetLatestTemplate(req.TemplateCode, req.TenantID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if existing != nil {
		return nil, fmt.Errorf("template already exists for templateId: %s, tenantId: %s. Use update to create new version",
			req.TemplateCode, req.TenantID)
	}

	// Run validations
	if err := validateTemplateConfig(req.Config); err != nil {
		return nil, err
	}

	// Marshal the config struct into JSON
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	idgenTemplateDB := &models.IDGenTemplateDB{
		ID:               uuid.New(),
		TemplateCode:     req.TemplateCode,
		TenantID:         req.TenantID,
		Version:          1, // Start with version 1
		Config:           configJSON,
		CreatedBy:        req.AuditDetails.CreatedBy,
		CreatedTime:      now,
		LastModifiedBy:   req.AuditDetails.CreatedBy,
		LastModifiedTime: now,
	}

	if err := s.repo.CreateTemplate(idgenTemplateDB); err != nil {
		return nil, err
	}

	if err := s.repo.CreateSequence(req.TenantID, req.TemplateCode, req.Config.Sequence.Start); err != nil {
		return nil, err
	}

	return idgenTemplateDB, nil
}

// UpdateTemplate creates a new version (incremented) instead of updating existing
func (s *IDGenService) UpdateTemplate(req *models.IDGenTemplate) (*models.IDGenTemplateDB, error) {
	// Get the latest version
	existing, err := s.repo.GetLatestTemplate(req.TemplateCode, req.TenantID)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		return nil, fmt.Errorf("template not found for templateId: %s, tenantId: %s. Use create first",
			req.TemplateCode, req.TenantID)
	}

	// Run validations
	if err := validateTemplateConfig(req.Config); err != nil {
		return nil, err
	}

	// Marshal the config struct into JSON
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, err
	}
	// Increment version
	newVersion := existing.Version + 1
	now := time.Now().Unix()

	idgenTemplateDB := &models.IDGenTemplateDB{
		ID:               uuid.New(),
		TemplateCode:     req.TemplateCode,
		TenantID:         req.TenantID,
		Version:          newVersion,
		Config:           configJSON,
		CreatedBy:        existing.CreatedBy,
		CreatedTime:      existing.CreatedTime,
		LastModifiedBy:   req.AuditDetails.LastModifiedBy,
		LastModifiedTime: now,
	}

	// Insert as new record
	if err := s.repo.CreateTemplate(idgenTemplateDB); err != nil {
		return nil, err
	}

	return idgenTemplateDB, nil
}

func (s *IDGenService) SearchTemplates(searchReq *models.IDGenTemplateSearch) ([]models.IDGenTemplateDB, error) {
	if searchReq.Version != "" {
		searchReq.VersionInt = parseVersion(searchReq.Version)
		if searchReq.VersionInt == 0 {
			return nil, fmt.Errorf("invalid version: %s", searchReq.Version)
		}
	}

	// Validation rule: version should always be mentioned with templateId
	if searchReq.VersionInt > 0 && searchReq.TemplateCode == "" {
		return nil, fmt.Errorf("invalid search: version filter requires templateCode")
	}

	return s.repo.SearchTemplates(searchReq)
}

func (s *IDGenService) DeleteTemplate(templateCode, tenantId, versionStr string) error {
	version := parseVersion(versionStr)
	if version == 0 {
		return fmt.Errorf("invalid version: %s", versionStr)
	}

	// Verify it exists
	if _, err := s.repo.GetByTenantIDTemplateCodeAndVersion(tenantId, templateCode, version); err != nil {
		return err
	}
	return s.repo.DeleteTemplate(tenantId, templateCode, version)
}

func (s *IDGenService) GenerateID(tenantId, templateCode string, variables map[string]string) (*models.GenerateIDResponse, error) {
	tmpl, err := s.repo.GetLatestTemplate(templateCode, tenantId)
	if err != nil {
		return nil, err
	}

	var config models.IDGenTemplateConfig
	if err := json.Unmarshal(tmpl.Config, &config); err != nil {
		return nil, err
	}

	parts := parseTemplate(config.Template)
	var output strings.Builder

	for _, part := range parts {
		if part.token == "" {
			output.WriteString(part.literal)
			continue
		}

		switch {
		case strings.HasPrefix(part.token, "DATE"):
			// Default format
			dateFormat := "20060102"

			if strings.Contains(part.token, ":") {
				keyword := strings.TrimPrefix(part.token, "DATE:")
				keyword = strings.ToLower(keyword)
				mapped, ok := dateKeywordMap[keyword]
				if !ok {
					return nil, fmt.Errorf("invalid date keyword: %s", keyword)
				}
				dateFormat = mapped
			}

			output.WriteString(time.Now().Format(dateFormat))

		case part.token == "SEQ":
			scopeKey := getScopeKey(config.Sequence.Scope, time.Now())
			if err := s.repo.EnsureScopeReset(tmpl.TenantID, templateCode, scopeKey, config.Sequence.Start); err != nil {
				return nil, err
			}

			seqVal, err := s.nextSeqWithRetry(tmpl.TenantID, templateCode)
			if err != nil {
				return nil, err
			}

			// Format sequence with custom padding character
			formatted := FormatSequence(int(seqVal), config.Sequence.Padding.Length, config.Sequence.Padding.Char)
			output.WriteString(formatted)

		case part.token == "RAND":
			randStr, err := randomString(config.Random.Length, config.Random.Charset)
			if err != nil {
				return nil, fmt.Errorf("failed to generate RAND: %v", err)
			}
			output.WriteString(randStr)

		default:
			if val, ok := variables[part.token]; ok {
				output.WriteString(val)
			} else {
				return nil, fmt.Errorf("missing variable: %s", part.token)
			}
		}
	}

	return &models.GenerateIDResponse{
		TenantID:     tmpl.TenantID,
		TemplateCode: tmpl.TemplateCode,
		Version:      fmt.Sprintf("v%d", tmpl.Version),
		ID:           output.String(),
	}, nil
}

func getScopeKey(scope string, t time.Time) string {
	switch scope {
	case "daily":
		return t.Format("2006-01-02")
	case "monthly":
		return t.Format("2006-01")
	case "yearly":
		return t.Format("2006")
	default:
		return "global"
	}
}

// FormatSequence formats a sequence number with left padding and custom character
func FormatSequence(seqVal int, padLength int, padChar string) string {
	if padLength <= 0 {
		return fmt.Sprintf("%d", seqVal)
	}
	if padChar == "" {
		padChar = "0"
	}

	numStr := fmt.Sprintf("%d", seqVal)
	padCount := max(padLength-len(numStr), 0)

	return strings.Repeat(padChar, padCount) + numStr
}

// randomString generates a random string of given length from a charset (supports ranges like "A-Z0-9")
func randomString(length int, charset string) (string, error) {
	if length <= 0 {
		return "", nil
	}

	// Expand charset ranges and validate
	expanded, err := expandCharset(charset)
	if err != nil {
		return "", fmt.Errorf("invalid charset: %v", err)
	}

	chars := []rune(expanded)
	if len(chars) == 0 {
		return "", fmt.Errorf("charset cannot be empty after expansion")
	}

	var out strings.Builder
	out.Grow(length) // preallocate memory

	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %v", err)
		}
		out.WriteRune(chars[idx.Int64()])
	}

	return out.String(), nil
}

// expandCharset expands simple ranges (like "A-Z0-9") to full characters
func expandCharset(charset string) (string, error) {
	if charset == "" {
		return "", fmt.Errorf("charset cannot be empty")
	}

	var result strings.Builder
	runes := []rune(charset)
	length := len(runes)

	for i := 0; i < length; i++ {
		if i+2 < length && runes[i+1] == '-' {
			start := runes[i]
			end := runes[i+2]

			if !validRange(start, end) {
				return "", fmt.Errorf("invalid range '%c-%c' in charset", start, end)
			}

			for r := start; r <= end; r++ {
				result.WriteRune(r)
			}
			i += 2 // skip the range
		} else {
			if runes[i] == '-' {
				return "", fmt.Errorf("invalid standalone '-' in charset")
			}
			result.WriteRune(runes[i])
		}
	}

	return result.String(), nil
}

// validRange checks if start and end are compatible (digits or same-case letters)
func validRange(start, end rune) bool {
	switch {
	case unicode.IsDigit(start) && unicode.IsDigit(end):
		return start <= end
	case unicode.IsUpper(start) && unicode.IsUpper(end):
		return start <= end
	case unicode.IsLower(start) && unicode.IsLower(end):
		return start <= end
	default:
		return false
	}
}

func validateTemplateConfig(config models.IDGenTemplateConfig) error {
	// 1. Validate DATE formats
	if err := validateDateTokens(config.Template); err != nil {
		return err
	}

	// 2. Validate sequence padding
	if err := validateSequencePadding(config.Sequence.Start, config.Sequence.Padding); err != nil {
		return err
	}

	// 3. Validate RAND charset
	if err := validateRandomCharset(config.Random); err != nil {
		return err
	}

	return nil
}

func validateDateTokens(tmpl string) error {
	parts := parseTemplate(tmpl)
	for _, part := range parts {
		if strings.HasPrefix(part.token, "DATE") {
			if strings.Contains(part.token, ":") {
				keyword := strings.TrimPrefix(part.token, "DATE:")
				keyword = strings.ToLower(keyword)
				if _, ok := dateKeywordMap[keyword]; !ok {
					return fmt.Errorf("invalid DATE format: %s", keyword)
				}
			}
		}
	}
	return nil
}

func validateSequencePadding(start int, pad models.PaddingConfig) error {
	startDigits := len(fmt.Sprintf("%d", start))
	if pad.Length < startDigits {
		return fmt.Errorf(
			"padding length (%d) is shorter than start value (%d digits)",
			pad.Length, startDigits,
		)
	}
	return nil
}

func validateRandomCharset(randCfg models.RandomConfig) error {
	if randCfg.Length <= 0 {
		return nil // optional, zero-length RAND is okay
	}
	if _, err := expandCharset(randCfg.Charset); err != nil {
		return fmt.Errorf("invalid RAND charset: %v", err)
	}
	return nil
}

func (s *IDGenService) nextSeqWithRetry(tenantId, templateCode string) (int64, error) {
	var seqVal int64
	const maxRetries = 5
	const backoff = 500 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		val, err := s.repo.NextSequenceValue(tenantId, templateCode)
		if err == nil {
			seqVal = val
			return seqVal, nil
		}
		log.Printf("Postgres unavailable, retry %d/%d: %v", i+1, maxRetries, err)
		time.Sleep(backoff)
	}
	return 0, fmt.Errorf("failed to fetch next sequence value for %s/%s after %d retries", tenantId, templateCode, maxRetries)
}
