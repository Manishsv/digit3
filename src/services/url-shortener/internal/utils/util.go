package utils

import (
	"net/url"
	"strings"
)

// IsValidURL returns true if URL is a valid full URL (http or https)
func IsValidURL(u string) bool {
	parsed, err := url.ParseRequestURI(u)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	return true
}
