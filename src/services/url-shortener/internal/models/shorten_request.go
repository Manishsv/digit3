package models

type ShortenRequest struct {
	URL       string `json:"url" binding:"required,url,max=2048"`
	ValidFrom int64  `json:"validFrom" default:"0"`
	ValidTill int64  `json:"validTill" default:"0"`
}
