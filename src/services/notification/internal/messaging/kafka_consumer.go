package messaging

import (
	"context"
	"encoding/json"
	"log"
	"notification/internal/config"
	"notification/internal/models"
	"notification/internal/service"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	emailService *service.EmailService
	smsService   *service.SMSService
	cfg          *config.Config
}

func NewKafkaConsumer(cfg *config.Config, emailSvc *service.EmailService, smsSvc *service.SMSService) *KafkaConsumer {
	log.Printf("Kafka consumer started. Listening to topics: EMAIL=%s, SMS=%s", cfg.EmailTopic, cfg.SMSTopic)
	return &KafkaConsumer{cfg: cfg, emailService: emailSvc, smsService: smsSvc}
}

func (c *KafkaConsumer) Start() error {
	go c.consumeEmail()
	go c.consumeSMS()
	return nil
}

func (c *KafkaConsumer) consumeEmail() {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: c.cfg.KafkaBrokers,
		Topic:   c.cfg.EmailTopic,
		GroupID: c.cfg.KafkaConsumerGroup,
	})
	defer r.Close()

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Println("error reading email message:", err)
			continue
		}
		log.Printf("Received Kafka EMAIL message: %s", string(m.Value))
		var req models.EmailRequest
		if err := json.Unmarshal(m.Value, &req); err != nil {
			log.Println("invalid email payload:", err)
			continue
		}
		c.emailService.SendEmail(context.Background(), &req)
	}
}

func (c *KafkaConsumer) consumeSMS() {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: c.cfg.KafkaBrokers,
		Topic:   c.cfg.SMSTopic,
		GroupID: c.cfg.KafkaConsumerGroup,
	})
	defer r.Close()

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Println("error reading sms message:", err)
			continue
		}
		log.Printf("Received Kafka SMS message: %s", string(m.Value))
		var req models.SMSRequest
		if err := json.Unmarshal(m.Value, &req); err != nil {
			log.Println("invalid sms payload:", err)
			continue
		}
		c.smsService.SendSMS(context.Background(), &req)
	}
}
