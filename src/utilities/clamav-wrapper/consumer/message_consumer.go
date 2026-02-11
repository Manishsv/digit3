// Package consumer provides interfaces and implementations for message consuming.
// It abstracts the underlying message broker technology (e.g., Kafka, RabbitMQ),
// allowing the application to interact with different messaging systems through a
// common interface.
package consumer

// MessageConsumer defines the interface for a message consumer.
// It provides a way to start consuming messages and to gracefully close the consumer.
type MessageConsumer interface {
	// StartConsumer begins listening for messages from the configured message broker.
	// It uses the handler function provided during its creation to process each message.
	// The method should block until an unrecoverable error occurs or the consumer is closed.
	StartConsumer() error

	// Close gracefully shuts down the message consumer, releasing any resources.
	// It should ensure that any in-flight message processing is completed if possible.
	Close() error
}
