package main

import (
	"log"
	"os"

	"billing-service/consumer"
)

func main() {
	amqpURI := os.Getenv("RABBITMQ_URI")
	if amqpURI == "" {
		amqpURI = os.Getenv("AMQP_URI")
	}
	if amqpURI == "" {
		log.Fatal("RABBITMQ_URI/AMQP_URI belum diset")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("BILLING_DB_DSN")
	}
	if dsn == "" {
		log.Fatal("DATABASE_URL/BILLING_DB_DSN belum diset")
	}

	consumer.StartBillingConsumer(amqpURI, dsn)
}
