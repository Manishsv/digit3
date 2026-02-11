package models

// Error represents a standard error response
type Error struct {
	Code        string   `json:"code"`
	Message     string   `json:"message"`
	Description string   `json:"description"`
	Params      []string `json:"params"`
}
