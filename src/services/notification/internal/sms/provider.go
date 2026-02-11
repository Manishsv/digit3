package sms

import (
	"notification/internal/models"
)

type SMSProvider interface {
	Send(mobileNumber, message string) []models.Error
}
