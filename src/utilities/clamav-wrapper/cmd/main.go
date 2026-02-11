package main

import (
	"log"
	"net/url"

	"clamav-wrapper/clamav"
	"clamav-wrapper/config"
	"clamav-wrapper/consumer"
	"clamav-wrapper/minio"
)

// processFileEvent handles the processing of a single file event.
func processFileEvent(bucketName string, objectKeyEncoded string) error {
	objectKey, err := url.QueryUnescape(objectKeyEncoded)
	if err != nil {
		log.Printf("Invalid object key: %v", err)
		return err
	}

	log.Printf("Processing file: %s from bucket: %s", objectKey, bucketName)

	file, size, err := minio.GetFileStreamWithSize(bucketName, objectKey)
	if err != nil {
		log.Printf("Failed to get file from MinIO: %v", err)
		return err
	}
	defer file.Close()

	isClean, err := clamav.Scan(file, size)
	if err != nil {
		log.Printf("ClamAV scan error: %v", err)
		return err
	}

	targetBucket := config.CleanBucket
	if !isClean {
		targetBucket = config.QuarantineBucket
		log.Printf("File %s in bucket %s is infected. Moving to quarantine.", objectKey, bucketName)
	} else {
		log.Printf("File %s in bucket %s is clean. Moving to clean bucket.", objectKey, bucketName)
	}

	if err := minio.CopyObject(bucketName, targetBucket, objectKey); err != nil {
		log.Printf("Failed to move file to %s bucket: %v", targetBucket, err)
		return err
	}

	if err := minio.DeleteObject(bucketName, objectKey); err != nil {
		log.Printf("Failed to delete original file %s from bucket %s: %v", objectKey, bucketName, err)
		return err
	}

	log.Printf("File %s processed and moved to %s bucket successfully.", objectKey, targetBucket)
	return nil
}

func main() {
	config.Init()
	minio.Init() // MinIO client needs to be initialized

	log.Printf("Initializing consumer for broker type: %s", config.MessageBrokerType)

	// Create an instance of the consumer factory
	consumerFactory := consumer.NewDefaultConsumerFactory()

	// Create the consumer using the factory
	// The handler (processFileEvent) is now passed at creation time.
	consumer, err := consumerFactory.CreateConsumer(config.MessageBrokerType, processFileEvent)
	if err != nil {
		log.Fatalf("Failed to create message consumer: %v", err)
	}

	// Defer the closing of the consumer right after it's successfully created and validated.
	defer func() {
		log.Println("Closing consumer...")
		if err := consumer.Close(); err != nil {
			log.Printf("Error closing consumer: %v", err)
		}
	}()

	log.Println("Starting consumer...")
	// Start the consumer. The handler is already configured.
	// This will block and continuously process messages.
	if err := consumer.StartConsumer(); err != nil {
		log.Fatalf("Consumer error: %v", err)
	}
}
