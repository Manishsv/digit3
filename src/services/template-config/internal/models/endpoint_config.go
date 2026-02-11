package models

// EndpointConfig represents external API endpoint configuration
type EndpointConfig struct {
	Base        string            `json:"base" binding:"required"`
	Path        string            `json:"path" binding:"required"`
	PathParams  map[string]string `json:"pathParams"`
	QueryParams map[string]string `json:"queryParams"`
}
