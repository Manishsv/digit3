package models

// APIMapping represents API enrichment rules
type APIMapping struct {
	Method          string            `json:"method" binding:"required"`
	Endpoint        EndpointConfig    `json:"endpoint" binding:"required"`
	ResponseMapping map[string]string `json:"responseMapping" binding:"required"`
}
