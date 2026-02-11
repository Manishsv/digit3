package messaging

import (
	"context"
	"encoding/json"
	"log"
	"notification/internal/config"
	"notification/internal/models"
	"notification/internal/service"

	"github.com/redis/go-redis/v9"
)

type RedisConsumer struct {
	emailService *service.EmailService
	smsService   *service.SMSService
	cfg          *config.Config
	client       *redis.Client
}

func NewRedisConsumer(cfg *config.Config, emailSvc *service.EmailService, smsSvc *service.SMSService) *RedisConsumer {
	client := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	return &RedisConsumer{cfg: cfg, emailService: emailSvc, smsService: smsSvc, client: client}
}

func (c *RedisConsumer) Start() error {
	log.Printf("Redis consumer started. Listening to channels: EMAIL=%s, SMS=%s", c.cfg.EmailTopic, c.cfg.SMSTopic)
	go c.consumeEmail()
	go c.consumeSMS()
	return nil
}

func (c *RedisConsumer) consumeEmail() {
	ctx := context.Background()
	sub := c.client.Subscribe(ctx, c.cfg.EmailTopic)
	ch := sub.Channel()
	for msg := range ch {
		log.Printf("Received Redis EMAIL message: %s", msg.Payload)
		var req models.EmailRequest
		if err := json.Unmarshal([]byte(msg.Payload), &req); err != nil {
			log.Println("invalid redis email payload:", err)
			continue
		}
		c.emailService.SendEmail(ctx, &req)
	}
}

func (c *RedisConsumer) consumeSMS() {
	ctx := context.Background()
	sub := c.client.Subscribe(ctx, c.cfg.SMSTopic)
	ch := sub.Channel()
	for msg := range ch {
		log.Printf("Received Redis SMS message: %s", msg.Payload)
		var req models.SMSRequest
		if err := json.Unmarshal([]byte(msg.Payload), &req); err != nil {
			log.Println("invalid redis sms payload:", err)
			continue
		}
		c.smsService.SendSMS(ctx, &req)
	}
}
