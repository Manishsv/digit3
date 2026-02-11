package handlers

import (
	"net/http"
	"url-shortener/internal/models"
	"url-shortener/internal/service"
	"url-shortener/internal/utils"

	"github.com/gin-gonic/gin"
)

type URLShortenerHandler struct {
	service *service.URLShortenerService
}

func NewURLShortenerHandler(service *service.URLShortenerService) *URLShortenerHandler {
	return &URLShortenerHandler{service: service}
}

func (h *URLShortenerHandler) ShortenURLHandler(c *gin.Context) {
	var request models.ShortenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Invalid request body",
			Description: err.Error(),
		})
		return
	}

	if !utils.IsValidURL(request.URL) {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Invalid URL",
			Description: "URL is not valid",
		})
		return
	}

	shortURL, err := h.service.ShortenURL(&request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:        "INTERNAL_ERROR",
			Message:     "Failed to shorten URL",
			Description: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"shortUrl": shortURL,
	})
}

func (h *URLShortenerHandler) RedirectURLHandler(c *gin.Context) {
	key := c.Param("key")
	url, err := h.service.RedirectURL(key)
	if err != nil {
		c.JSON(http.StatusNotFound, models.Error{
			Code:        "NOT_FOUND",
			Message:     "URL not found",
			Description: err.Error(),
		})
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, url)
}
