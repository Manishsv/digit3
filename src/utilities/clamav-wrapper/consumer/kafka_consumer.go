package consumer

import (
	"context"
	"encoding/json"
	"fmt" // Added for fmt.Errorf
	"log"

	"github.com/segmentio/kafka-go"

	"clamav-wrapper/config"
	"clamav-wrapper/models"
)

// KafkaConsumer implements the MessageConsumer interface for Apache Kafka.
// It handles the connection to Kafka, message consumption, and deserialization.
type KafkaConsumer struct {
	Reader  *kafka.Reader
	handler func(bucketName string, objectKeyEncoded string) error
}

// NewKafkaConsumer creates and configures a new KafkaConsumer.
// It initializes a Kafka reader based on configuration from config.KafkaCfg
// and stores the provided handler function.
// Returns the configured KafkaConsumer or an error if initialization fails (e.g. nil handler).
func NewKafkaConsumer(cfg config.KafkaConfig, handlerFunc func(bucketName string, objectKeyEncoded string) error) (*KafkaConsumer, error) {
	if handlerFunc == nil {
		return nil, fmt.Errorf("handler function cannot be nil for KafkaConsumer")
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.Topic,
		GroupID: cfg.ConsumerGroupID,
	})
	return &KafkaConsumer{Reader: r, handler: handlerFunc}, nil
}

// StartConsumer begins consuming messages from the Kafka topic.
// It continuously reads messages, deserializes them into models.KakfaEvent,
// and then passes them to the handler function stored in the KafkaConsumer.
// This method will block until a read error occurs, the context is cancelled
// (which can be triggered by closing the reader via the Close method), or if the handler is not set.
func (kc *KafkaConsumer) StartConsumer() error {
	if kc.handler == nil {
		return fmt.Errorf("KafkaConsumer's handler is not set")
	}

	log.Printf("Subscribed to kafka topic: %s", config.KafkaCfg.Topic)

	for {
		m, err := kc.Reader.ReadMessage(context.Background())
		if err != nil {
			// If the reader is closed, ReadMessage will return an error.
			// This is the expected way to stop the consumer.
			log.Printf("Kafka read error: %v. This might indicate the consumer is closing.", err)
			return err // Return error to signal consumer stop or failure
		}

		var event models.KafkaEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("Invalid event message: %v. Skipping message.", err)
			continue // Skip malformed messages
		}

		for _, record := range event.Records {
			// Call the stored handler with the received event
			if err := kc.handler(record.S3.Bucket.Name, record.S3.Object.Key); err != nil {
				log.Printf("Error processing event: %v. Continuing to next message.", err)
			}
		}
	}
}

// Close gracefully shuts down the Kafka consumer by closing the underlying Kafka reader.
// This will cause the StartConsumer loop to exit.
func (kc *KafkaConsumer) Close() error {
	if kc.Reader != nil {
		log.Println("Closing Kafka consumer reader.")
		return kc.Reader.Close()
	}
	return nil
}
