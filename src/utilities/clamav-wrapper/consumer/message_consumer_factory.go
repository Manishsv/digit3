package consumer

import (
	"clamav-wrapper/config" // Added to access config.RedisCfg
	"fmt"
)

// MessageConsumerFactory defines an interface for creating instances of MessageConsumer.
// This allows for different implementations of consumers (e.g., Kafka, Redis)
// to be created based on configuration.
type MessageConsumerFactory interface {
	// CreateConsumer constructs a new MessageConsumer based on the specified brokerType.
	// It takes a handler function that will be invoked for each message received by the consumer.
	// Returns the configured MessageConsumer or an error if the brokerType is unsupported
	// or if there's an issue during consumer initialization.
	CreateConsumer(brokerType string, handler func(bucketName string, objectKeyEncoded string) error) (MessageConsumer, error)
}

// DefaultConsumerFactory is a concrete implementation of MessageConsumerFactory.
// It can create Kafka consumers. Support for other types like Redis can be added here.
type DefaultConsumerFactory struct{}

// NewDefaultConsumerFactory creates a new instance of DefaultConsumerFactory.
func NewDefaultConsumerFactory() *DefaultConsumerFactory {
	return &DefaultConsumerFactory{}
}

// CreateConsumer creates a message consumer based on the brokerType.
// It supports "kafka" and "redis" broker types.
func (f *DefaultConsumerFactory) CreateConsumer(brokerType string, handler func(bucketName string, objectKeyEncoded string) error) (MessageConsumer, error) {
	if handler == nil {
		return nil, fmt.Errorf("handler function cannot be nil for CreateConsumer")
	}
	switch brokerType {
	case "kafka":
		// NewKafkaConsumer now takes the handler and returns (consumer, error)
		consumer, err := NewKafkaConsumer(config.KafkaCfg, handler)
		if err != nil {
			return nil, fmt.Errorf("error creating Kafka consumer: %w", err)
		}
		return consumer, nil
	case "redis":
		consumer, err := NewRedisConsumer(config.RedisCfg, handler)
		if err != nil {
			return nil, fmt.Errorf("error creating Redis consumer: %w", err)
		}
		return consumer, nil
	default:
		return nil, fmt.Errorf("unsupported message broker type: %s", brokerType)
	}
}
