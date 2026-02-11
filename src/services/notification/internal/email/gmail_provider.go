package email

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"notification/internal/config"
	"notification/internal/models"
	"strings"
)

type GmailProvider struct {
	config *config.Config
}

func NewGmailProvider(config *config.Config) *GmailProvider {
	return &GmailProvider{
		config: config,
	}
}

func (p *GmailProvider) Send(to []mail.Address, subject, body string, isHTML bool, attachments []Attachment) []models.Error {
	from := mail.Address{Name: p.config.SMTPFromName, Address: p.config.SMTPFromAddress}

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = from.String()
	headers["To"] = p.formatAddressList(to)
	headers["Subject"] = subject

	// Setup message
	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	// Create a new multipart writer
	mw := multipart.NewWriter(&msg)
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", mw.Boundary()))
	msg.WriteString("\r\n")

	// Add body part using textproto.MIMEHeader
	bodyHeader := textproto.MIMEHeader{}
	var contentType string
	if isHTML {
		contentType = "text/html; charset=UTF-8"
	} else {
		contentType = "text/plain; charset=UTF-8"
	}
	bodyHeader.Set("Content-Type", contentType)

	bodyPart, err := mw.CreatePart(bodyHeader)
	if err != nil {
		return []models.Error{{
			Code:        "CREATE_BODY_PART_ERROR",
			Message:     "Failed to create body part",
			Description: err.Error(),
		}}
	}

	_, err = bodyPart.Write([]byte(body))
	if err != nil {
		return []models.Error{{
			Code:        "WRITE_BODY_PART_ERROR",
			Message:     "Failed to write body part",
			Description: err.Error(),
		}}
	}

	// Add attachments using textproto.MIMEHeader
	for _, attachment := range attachments {
		attachmentHeader := textproto.MIMEHeader{}
		attachmentHeader.Set("Content-Type", "application/octet-stream")
		attachmentHeader.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, attachment.Filename))

		attachmentPart, err := mw.CreatePart(attachmentHeader)
		if err != nil {
			return []models.Error{{
				Code:        "CREATE_ATTACHMENT_PART_ERROR",
				Message:     "Failed to create attachment part",
				Description: err.Error(),
			}}
		}
		_, err = attachmentPart.Write(attachment.Data)
		if err != nil {
			return []models.Error{{
				Code:        "WRITE_ATTACHMENT_ERROR",
				Message:     "Failed to write attachment",
				Description: err.Error(),
			}}
		}
	}

	err = mw.Close()
	if err != nil {
		return []models.Error{{
			Code:        "CLOSE_MULTIPART_WRITER_ERROR",
			Message:     "Failed to close multipart writer",
			Description: err.Error(),
		}}
	}

	// Authenticate and send email
	auth := smtp.PlainAuth("", p.config.SMTPUsername, p.config.SMTPPassword, p.config.SMTPHost)
	err = smtp.SendMail(fmt.Sprintf("%s:%d", p.config.SMTPHost, p.config.SMTPPort), auth, from.Address, p.toAddressList(to), msg.Bytes())
	if err != nil {
		return []models.Error{{
			Code:        "SEND_MAIL_ERROR",
			Message:     "Failed to send email",
			Description: err.Error(),
		}}
	}

	return nil
}

func (p *GmailProvider) formatAddressList(list []mail.Address) string {
	var addresses []string
	for _, addr := range list {
		addresses = append(addresses, addr.String())
	}
	return strings.Join(addresses, ", ")
}

func (p *GmailProvider) toAddressList(list []mail.Address) []string {
	var addresses []string
	for _, addr := range list {
		addresses = append(addresses, addr.Address)
	}
	return addresses
}
