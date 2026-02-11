package kafka

import (
	"context"
	"github.com/Shopify/sarama"
)

// MessageHandler defines the interface for handling Kafka messages
type MessageHandler interface {
	// Handle processes a Kafka message
	Handle(ctx context.Context, message []byte) error
}

// Consumer defines the interface for Kafka message consumption
type Consumer interface {
	// Start begins consuming messages from the specified topics
	Start(ctx context.Context, topics []string) error
	// Stop stops consuming messages
	Stop() error
}

// KafkaConsumer implements the Consumer interface
type KafkaConsumer struct {
	consumer sarama.ConsumerGroup
	handler  MessageHandler
	groupID  string
}

// NewKafkaConsumer creates a new Kafka consumer
func NewKafkaConsumer(brokers []string, groupID string, handler MessageHandler) (*KafkaConsumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumer, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, err
	}

	return &KafkaConsumer{
		consumer: consumer,
		handler:  handler,
		groupID:  groupID,
	}, nil
}

// Start implements Consumer.Start
func (c *KafkaConsumer) Start(ctx context.Context, topics []string) error {
	consumerGroup := &consumerGroupHandler{
		handler: c.handler,
	}

	for {
		err := c.consumer.Consume(ctx, topics, consumerGroup)
		if err != nil {
			return err
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

// Stop implements Consumer.Stop
func (c *KafkaConsumer) Stop() error {
	return c.consumer.Close()
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	handler MessageHandler
}

// Setup is called when the consumer is assigned to a new consumer group
func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is called when the consumer is removed from a consumer group
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim processes messages from a claim
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		err := h.handler.Handle(session.Context(), message.Value)
		if err != nil {
			// Log error but continue processing
			continue
		}
		session.MarkMessage(message, "")
	}
	return nil
} 