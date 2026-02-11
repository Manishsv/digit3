package dtos

// MessageResponse is a simplified message structure for responses
type MessageResponse struct {
	UUID    string `json:"uuid"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Module  string `json:"module"`
	Locale  string `json:"locale"`
}

// UpsertMessagesResponse represents the response for the upsert localization messages API
type UpsertMessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
}

// SearchMessagesResponse represents the response for the search localization messages API
type SearchMessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
}

// CreateMessagesResponse represents the response for the create localization messages API
type CreateMessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
}

// UpdateMessagesResponse represents the response for the update localization messages API
type UpdateMessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
}

// DeleteMessagesResponse represents the response for the delete localization messages API
type DeleteMessagesResponse struct {
	Success bool `json:"success"`
}

// CacheBustResponse represents the response for the cache bust API
type CacheBustResponse struct {
	Message string `json:"message,omitempty"`
	Success bool   `json:"success"`
}
