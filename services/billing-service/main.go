package main

import (
	"log"
	"os"

	"billing-service/consumer"
)

func main() {
	amqpURI := os.Getenv("AMQP_URI")
	if amqpURI == "" {
		log.Fatal("AMQP_URI belum diset (contoh: amqp://USER:PASS@localhost:5672/medcore_vhost)")
	}
	dsn := os.Getenv("BILLING_DB_DSN")
	if dsn == "" {
		log.Fatal("BILLING_DB_DSN belum diset (contoh: USER:PASS@tcp(localhost:3306)/medcore_billing_db)")
	}
	consumer.StartBillingConsumer(amqpURI, dsn)
}
