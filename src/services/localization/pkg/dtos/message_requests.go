package dtos

// SearchMessagesRequest defines the request body for searching messages
type SearchMessagesRequest struct {
	Module string   `json:"module,omitempty"`
	Locale string   `json:"locale,omitempty"`
	Codes  []string `json:"codes,omitempty"`
}

// CreateMessagesRequest defines the request body for creating multiple messages
type CreateMessagesRequest struct {
	Messages []Message `json:"messages"`
}

// UpdateMessagesRequest defines the request body for updating messages for a specific module
type UpdateMessagesRequest struct {
	Messages []Message `json:"messages"`
}

// UpsertMessagesRequest defines the request body for upserting multiple messages
type UpsertMessagesRequest struct {
	Messages []Message `json:"messages"`
}

// DeleteMessagesRequest defines the request body for deleting multiple messages
type DeleteMessagesRequest struct {
	UUIDs []string `json:"uuids"`
}

// Message represents a single localization message
type Message struct {
	UUID    string `json:"uuid,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Module  string `json:"module"`
	Locale  string `json:"locale"`
}

// MessageIdentity uniquely identifies a message to be deleted
type MessageIdentity struct {
	Module string `json:"module"`
	Locale string `json:"locale"`
	Code   string `json:"code"`
}

// UpdateMessage represents a message to be updated
type UpdateMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// DeleteMessage represents a message to be deleted
type DeleteMessage struct {
	Code   string `json:"code"`
	Module string `json:"module"`
	Locale string `json:"locale"`
}

// FindMissingMessagesRequest represents the request to find missing messages
type FindMissingMessagesRequest struct {
	Locales []string `json:"locales"`
}
