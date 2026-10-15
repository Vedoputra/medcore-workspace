package consumer

import (
	"context"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
)

// EventEnvelope merepresentasikan bagian event
// ConsultationCompletedEvent yang dibutuhkan Pharmacy.
type EventEnvelope struct {
	EventID   string `json:"event_id"`
	EventName string `json:"event_name"`

	Payload struct {
		AppointmentID string `json:"appointment_id"`

		Prescription struct {
			PrescriptionID string `json:"prescription_id"`

			Items []struct {
				DrugCode string `json:"drug_code"`
				Quantity int    `json:"quantity"`
				Dosage   string `json:"dosage"`
			} `json:"items"`
		} `json:"prescription"`
	} `json:"payload"`
}

// StartPrescriptionConsumer menjalankan RabbitMQ worker
// yang menerima event konsultasi dan membaca data resep.
func StartPrescriptionConsumer(amqpURI string, mongoClient *mongo.Client) {

	// 1. Hubungkan Pharmacy Service ke RabbitMQ.
	conn, err := amqp.Dial(amqpURI)
	if err != nil {
		log.Fatalf(
			"[Pharmacy Consumer] Gagal koneksi broker: %v",
			err,
		)
	}
	defer conn.Close()

	// 2. Buat AMQP channel.
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf(
			"[Pharmacy Consumer] Gagal membuka channel: %v",
			err,
		)
	}
	defer ch.Close()

	// 3. Konfigurasi Pharmacy Queue.
	//
	// Penyesuaian untuk topology D:
	// Gunakan DLX yang sudah disepakati kelompok.
	//
	// Atribut ini harus sama dengan deklarasi queue
	// yang sudah ada pada RabbitMQ.
	queueArgs := amqp.Table{
		"x-dead-letter-exchange": "medcore.deadletter.exchange",
	}

	q, err := ch.QueueDeclare(
		"pharmacy.prescription.order.q",
		true,  // Durable
		false, // Auto-delete
		false, // Exclusive
		false, // No-wait
		queueArgs,
	)
	if err != nil {
		log.Fatalf(
			"[Pharmacy Consumer] Gagal deklarasi queue: %v",
			err,
		)
	}

	// 4. Binding Pharmacy Queue ke Topic Exchange.
	err = ch.QueueBind(
		q.Name,
		"clinical.consultation.*",
		"medcore.clinical.events",
		false,
		nil,
	)
	if err != nil {
		log.Fatalf(
			"[Pharmacy Consumer] Gagal binding queue: %v",
			err,
		)
	}

	// 5. Batasi satu pesan outstanding per consumer.
	err = ch.Qos(1, 0, false)
	if err != nil {
		log.Fatalf(
			"[Pharmacy Consumer] Gagal set QoS: %v",
			err,
		)
	}

	// 6. Subscribe menggunakan manual acknowledgement.
	msgs, err := ch.Consume(
		q.Name,
		"pharmacy-worker",
		false, // autoAck = false
		false, // Exclusive
		false, // No-local
		false, // No-wait
		nil,
	)
	if err != nil {
		log.Fatalf(
			"[Pharmacy Consumer] Gagal subscribe: %v",
			err,
		)
	}

	log.Printf(
		"[Pharmacy Consumer] Menunggu pesan resep pada antrean: %s",
		q.Name,
	)

	// 7. Proses pesan satu per satu (lihat processor.go).
	for d := range msgs {
		processPrescription(context.Background(), mongoClient, ch, d)
	}

	log.Println("[Pharmacy Consumer] Consumer berhenti.")
}
