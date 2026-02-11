package service

import (
	"context"
	"net/mail"
	"notification/internal/config"
	"notification/internal/email"
	"notification/internal/models"
	"notification/internal/utils"
	"os"
)

type EmailService struct {
	templateService *TemplateService
	filestoreClient *utils.FilestoreClient
	emailProvider   email.EmailProvider
	config          *config.Config
}

func NewEmailService(templateService *TemplateService, emailProvider email.EmailProvider, cfg *config.Config) *EmailService {
	// Create filestore client with base path and endpoint from config
	filestoreClient := utils.NewFilestoreClient(cfg.FilestoreHost, cfg.FilestorePath)

	return &EmailService{
		templateService: templateService,
		filestoreClient: filestoreClient,
		emailProvider:   emailProvider,
		config:          cfg,
	}
}

func (s *EmailService) SendEmail(ctx context.Context, req *models.EmailRequest) []models.Error {
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

	if previewResp.Type != models.TemplateTypeEmail {
		return []models.Error{{
			Code:        "INVALID_TEMPLATE_TYPE",
			Message:     "Invalid template type",
			Description: "Only email templates are supported",
		}}
	}

	// 2. Fetch attachments
	var attachments []email.Attachment
	for _, fileID := range req.Attachments {
		file, err := s.filestoreClient.GetFile(ctx, fileID, req.TenantID)
		if err != nil {
			return []models.Error{{
				Code:        "FILESTORE_ERROR",
				Message:     "Filestore Error",
				Description: err.Error(),
			}}
		}
		fdata, err := os.ReadFile(file.Path)
		if err != nil {
			return []models.Error{{
				Code:        "READ_FILE_ERROR",
				Message:     "Failed to read file from disk",
				Description: err.Error(),
			}}
		}
		// Ensure temp file is cleaned up after reading
		defer os.Remove(file.Path)
		attachments = append(attachments, email.Attachment{
			Filename: file.Filename,
			Data:     fdata,
		})
	}

	// 3. Convert email strings to mail.Address
	var to []mail.Address
	for _, emailStr := range req.EmailIds {
		to = append(to, mail.Address{Address: emailStr})
	}

	// 4. Send the email
	errors = s.emailProvider.Send(to, previewResp.RenderedSubject, previewResp.RenderedContent, previewResp.IsHTML, attachments)
	if len(errors) > 0 {
		return errors
	}
	return nil
}
