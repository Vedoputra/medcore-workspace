package consumer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

func StartBillingConsumer(amqpURI, mysqlDSN string) {
	db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatalf("[Billing Consumer] Gagal membuka database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("[Billing Consumer] Gagal koneksi database: %v", err)
	}
	// Tabel deduplikasi (idempotensi)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS processed_events (
		event_id VARCHAR(64) PRIMARY KEY,
		processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB`)
	if err != nil {
		log.Fatalf("[Billing Consumer] Gagal membuat tabel processed_events: %v", err)
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
	args := amqp.Table{"x-dead-letter-exchange": "medcore.deadletter.billing.exchange"}
	q, err := ch.QueueDeclare("billing.invoice.q", true, false, false, false, args)
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
		handle(context.Background(), db, d)
	}
}

func handle(ctx context.Context, db *sql.DB, d amqp.Delivery) {
	var evt EventEnvelope
	if err := json.Unmarshal(d.Body, &evt); err != nil || evt.EventID == "" {
		log.Printf("[Billing Consumer] Format pesan tidak valid, NACK tanpa requeue: %v", err)
		d.Nack(false, false)
		return
	}
	fee := evt.Payload.BillingSummary.ConsultationFee
	if fee <= 0 {
		log.Printf("[Billing Consumer] Poison Message! consultation_fee tidak valid: %.2f. NACK ke DLQ.", fee)
		d.Nack(false, false)
		return
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("[Billing Consumer] Gagal memulai transaksi: %v. Requeue.", err)
		d.Nack(false, true)
		return
	}
	defer tx.Rollback()

	// Idempotensi: INSERT gagal jika event_id sudah ada
	res, err := tx.ExecContext(ctx, "INSERT IGNORE INTO processed_events (event_id) VALUES (?)", evt.EventID)
	if err != nil {
		log.Printf("[Billing Consumer] Gagal cek deduplikasi: %v. Requeue.", err)
		d.Nack(false, true)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		log.Printf("[Deduplication Guard] Event %s terdeteksi duplikat! Melewati pembuatan invoice dan langsung ACK.", evt.EventID)
		d.Ack(false)
		return
	}

	invoiceID := uuid.New().String()
	if _, err = tx.ExecContext(ctx,
		"INSERT INTO invoices (invoice_id, patient_id, appointment_id, total_amount, payment_status) VALUES (?, ?, ?, ?, 'UNPAID')",
		invoiceID, evt.Payload.PatientID, evt.Payload.AppointmentID, fee); err != nil {
		log.Printf("[Billing Consumer] Gagal menyimpan invoice: %v. Requeue.", err)
		d.Nack(false, true)
		return
	}
	if _, err = tx.ExecContext(ctx,
		"INSERT INTO invoice_items (item_id, invoice_id, description, amount) VALUES (?, ?, ?, ?)",
		uuid.New().String(), invoiceID, "Jasa Konsultasi", fee); err != nil {
		log.Printf("[Billing Consumer] Gagal menyimpan item invoice: %v. Requeue.", err)
		d.Nack(false, true)
		return
	}
	if err = tx.Commit(); err != nil {
		log.Printf("[Billing Consumer] Commit gagal: %v. Requeue.", err)
		d.Nack(false, true)
		return
	}

	d.Ack(false)
	log.Printf("[Billing Consumer] Draf tagihan %s dibuat untuk appointment %s (Rp %s). Event %s di-ACK.",
		invoiceID, evt.Payload.AppointmentID, fmt.Sprintf("%.2f", fee), evt.EventID)
}
