package service

import (
	"context"
	"notification/internal/config"
	"notification/internal/models"
	"notification/internal/sms"
)

type SMSService struct {
	templateService *TemplateService
	smsProvider     sms.SMSProvider
	config          *config.Config
}

func NewSMSService(templateService *TemplateService, smsProvider sms.SMSProvider, cfg *config.Config) *SMSService {
	return &SMSService{
		templateService: templateService,
		smsProvider:     smsProvider,
		config:          cfg,
	}
}

func (s *SMSService) SendSMS(ctx context.Context, req *models.SMSRequest) []models.Error {
	// 1. Render the template
	previewReq := &models.TemplatePreviewRequest{
		TemplateID: req.TemplateID,
		TenantID:   req.TenantID,
		Version:    req.Version,
		Payload:    req.Payload,
		Enrich:     req.Enrich,
	}
	previewResp, errors := s.templateService.Preview(previewReq)
	if len(errors) > 0 {
		return errors
	}

	if previewResp.Type != models.TemplateTypeSMS {
		return []models.Error{{
			Code:        "INVALID_TEMPLATE_TYPE",
			Message:     "Invalid template type",
			Description: "Only SMS templates are supported",
		}}
	}

	// 2. Send the sms
	for _, mobileNumber := range req.MobileNumbers {
		errors := s.smsProvider.Send(mobileNumber, previewResp.RenderedContent)
		if len(errors) > 0 {
			return errors
		}
	}

	return nil
}
