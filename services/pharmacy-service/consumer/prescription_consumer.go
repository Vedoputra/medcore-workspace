package consumer

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
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
func StartPrescriptionConsumer(amqpURI string) {

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

	// 7. Proses pesan satu per satu.
	for d := range msgs {

		var evt EventEnvelope

		// Decode JSON.
		if err := json.Unmarshal(d.Body, &evt); err != nil {
			log.Printf(
				"[Pharmacy Consumer] Format pesan tidak valid: %v",
				err,
			)

			// Permanent rejection.
			// Pesan masuk DLQ jika DLX dikonfigurasi dengan benar.
			if err := d.Nack(false, false); err != nil {
				log.Printf(
					"[Pharmacy Consumer] Gagal NACK: %v",
					err,
				)
			}

			continue
		}

		// 8. Simulasikan pemrosesan resep.
		log.Println("---------------------------------------------------------")
		log.Println("[Pharmacy Consumer] Menerima Resep Baru!")
		log.Printf(" Event ID        : %s", evt.EventID)
		log.Printf(" Appointment ID  : %s", evt.Payload.AppointmentID)
		log.Printf(
			" Prescription ID : %s",
			evt.Payload.Prescription.PrescriptionID,
		)

		for _, item := range evt.Payload.Prescription.Items {
			log.Printf(
				" -> Item: %s | Qty: %d",
				item.DrugCode,
				item.Quantity,
			)
		}

		log.Println("---------------------------------------------------------")

		// 9. ACK setelah pemrosesan simulasi selesai.
		if err := d.Ack(false); err != nil {
			log.Printf(
				"[Pharmacy Consumer] Gagal ACK event %s: %v",
				evt.EventID,
				err,
			)
			continue
		}

		log.Printf(
			"[Pharmacy Consumer] Event %s berhasil di-ACK.",
			evt.EventID,
		)
	}

	log.Println("[Pharmacy Consumer] Consumer berhenti.")
}
