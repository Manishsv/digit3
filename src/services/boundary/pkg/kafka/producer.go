package kafka

import (
	"context"
	"encoding/json"
	"github.com/Shopify/sarama"
)

// Producer defines the interface for Kafka message production
type Producer interface {
	// Push sends a message to a Kafka topic
	Push(ctx context.Context, topic string, message interface{}) error
}

// KafkaProducer implements the Producer interface
type KafkaProducer struct {
	producer sarama.SyncProducer
}

// NewKafkaProducer creates a new Kafka producer
func NewKafkaProducer(brokers []string) (*KafkaProducer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return &KafkaProducer{
		producer: producer,
	}, nil
}

// Push implements Producer.Push
func (p *KafkaProducer) Push(ctx context.Context, topic string, message interface{}) error {
	// Marshal message to JSON
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Create Kafka message
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(msgBytes),
	}

	// Send message
	_, _, err = p.producer.SendMessage(msg)
	return err
}

// Close closes the Kafka producer
func (p *KafkaProducer) Close() error {
	return p.producer.Close()
} 