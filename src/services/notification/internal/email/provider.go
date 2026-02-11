package email

import (
	"net/mail"
	"notification/internal/models"
)

type Attachment struct {
	Filename string
	Data     []byte
}

type EmailProvider interface {
	Send(to []mail.Address, subject, body string, isHTML bool, attachments []Attachment) []models.Error
}
