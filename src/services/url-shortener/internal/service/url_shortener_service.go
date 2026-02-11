package service

import (
	"fmt"
	"strings"

	"url-shortener/internal/config"
	"url-shortener/internal/models"
	"url-shortener/internal/repository"
)

type URLShortenerService struct {
	repo *repository.URLShortenerRepository
	cfg  *config.Config
}

func NewURLShortenerService(repo *repository.URLShortenerRepository, cfg *config.Config) *URLShortenerService {
	return &URLShortenerService{repo: repo, cfg: cfg}
}

func (s *URLShortenerService) ShortenURL(request *models.ShortenRequest) (string, error) {

	// Short key generation
	shortKey, err := s.repo.GetOrCreateShortKey(*request)
	if err != nil {
		return "", err
	}

	shortURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(s.cfg.HostName, "/"), shortKey)
	if s.cfg.ServerContextPath != "" {
		shortURL = fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(s.cfg.HostName, "/"), strings.Trim(s.cfg.ServerContextPath, "/"), shortKey)
	}

	return shortURL, nil
}

func (s *URLShortenerService) RedirectURL(shortKey string) (string, error) {
	longURL, err := s.repo.GetURL(shortKey)
	if err != nil {
		return "", err
	}
	return longURL, nil
}
