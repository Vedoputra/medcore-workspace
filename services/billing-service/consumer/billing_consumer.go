package consumer

import (
	"database/sql"
	"encoding/json"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type EventEnvelope struct {
	EventID string `json:"event_id"`
	Payload struct {
		AppointmentID  string `json:"appointment_id"`
		PatientID      string `json:"patient_id"`
		BillingSummary struct {
			ConsultationFee float64 `json:"consultation_fee"`
		} `json:"billing_summary"`
	} `json:"payload"`
}

// StartBillingConsumer menerima event konsultasi selesai dan mencatat draf tagihan.
func StartBillingConsumer(amqpURI, mysqlDSN string) {
	db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatalf("[Billing Consumer] Gagal membuka database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("[Billing Consumer] Gagal koneksi database: %v", err)
	}

	conn, err := amqp.Dial(amqpURI)
	if err != nil {
		log.Fatalf("[Billing Consumer] Gagal koneksi broker: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("[Billing Consumer] Gagal membuka channel: %v", err)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare("medcore.clinical.events", "topic", true, false, false, false, nil); err != nil {
		log.Fatalf("[Billing Consumer] Gagal deklarasi exchange: %v", err)
	}

	q, err := ch.QueueDeclare("billing.invoice.q", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("[Billing Consumer] Gagal deklarasi queue: %v", err)
	}
	if err := ch.QueueBind(q.Name, "clinical.consultation.completed", "medcore.clinical.events", false, nil); err != nil {
		log.Fatalf("[Billing Consumer] Gagal binding queue: %v", err)
	}
	if err := ch.Qos(1, 0, false); err != nil {
		log.Fatalf("[Billing Consumer] Gagal set QoS: %v", err)
	}

	msgs, err := ch.Consume(q.Name, "billing-worker", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("[Billing Consumer] Gagal subscribe: %v", err)
	}
	log.Printf("[Billing Consumer] Menunggu event konsultasi pada antrean: %s", q.Name)

	for d := range msgs {
		var evt EventEnvelope
		if err := json.Unmarshal(d.Body, &evt); err != nil {
			log.Printf("[Billing Consumer] Format pesan tidak valid, NACK tanpa requeue: %v", err)
			d.Nack(false, false)
			continue
		}

		fee := evt.Payload.BillingSummary.ConsultationFee
		invoiceID := uuid.New().String()

		_, err := db.Exec(
			"INSERT INTO invoices (invoice_id, patient_id, appointment_id, total_amount, payment_status) VALUES (?, ?, ?, ?, 'UNPAID')",
			invoiceID, evt.Payload.PatientID, evt.Payload.AppointmentID, fee)
		if err == nil {
			_, err = db.Exec(
				"INSERT INTO invoice_items (item_id, invoice_id, description, amount) VALUES (?, ?, ?, ?)",
				uuid.New().String(), invoiceID, "Jasa Konsultasi", fee)
		}
		if err != nil {
			log.Printf("[Billing Consumer] Gagal menyimpan tagihan: %v. Requeue.", err)
			d.Nack(false, true)
			continue
		}

		d.Ack(false)
		log.Printf("[Billing Consumer] Draf tagihan %s dibuat untuk appointment %s (Rp %.2f). Event %s di-ACK.",
			invoiceID, evt.Payload.AppointmentID, fee, evt.EventID)
	}
}
