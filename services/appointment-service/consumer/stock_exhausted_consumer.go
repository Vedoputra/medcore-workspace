package consumer

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

type stockExhaustedEvent struct {
	EventID string `json:"event_id"`
	Payload struct {
		AppointmentID string `json:"appointment_id"`
		DrugCode      string `json:"drug_code"`
		Requested     int    `json:"requested_quantity"`
		CurrentStock  int    `json:"current_stock"`
	} `json:"payload"`
}

// StartStockExhaustedConsumer mendengarkan event kompensasi
// pharmacy.stock.exhausted dan mengubah status appointment menjadi
// PENDING_MEDICATION_SUBSTITUTION.
func StartStockExhaustedConsumer(amqpURI, dbDSN string) {
	ctx := context.Background()

	db, err := pgx.Connect(ctx, dbDSN)
	if err != nil {
		log.Fatalf("[Appointment Consumer] Gagal koneksi database: %v", err)
	}
	defer db.Close(ctx)

	conn, err := amqp.Dial(amqpURI)
	if err != nil {
		log.Fatalf("[Appointment Consumer] Gagal koneksi broker: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("[Appointment Consumer] Gagal membuka channel: %v", err)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare("medcore.clinical.events", "topic", true, false, false, false, nil); err != nil {
		log.Fatalf("[Appointment Consumer] Gagal deklarasi exchange: %v", err)
	}
	q, err := ch.QueueDeclare("appointment.stock.exhausted.q", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("[Appointment Consumer] Gagal deklarasi queue: %v", err)
	}
	if err := ch.QueueBind(q.Name, "pharmacy.stock.exhausted", "medcore.clinical.events", false, nil); err != nil {
		log.Fatalf("[Appointment Consumer] Gagal binding queue: %v", err)
	}
	if err := ch.Qos(1, 0, false); err != nil {
		log.Fatalf("[Appointment Consumer] Gagal set QoS: %v", err)
	}
	msgs, err := ch.Consume(q.Name, "appointment-worker", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("[Appointment Consumer] Gagal subscribe: %v", err)
	}
	log.Printf("[Appointment Consumer] Menunggu event stok habis pada antrean: %s", q.Name)

	for d := range msgs {
		var evt stockExhaustedEvent
		if err := json.Unmarshal(d.Body, &evt); err != nil {
			log.Printf("[Appointment Consumer] Format tidak valid, NACK tanpa requeue: %v", err)
			d.Nack(false, false)
			continue
		}

		tag, err := db.Exec(ctx,
			"UPDATE appointments SET consultation_status = 'PENDING_MEDICATION_SUBSTITUTION' WHERE appointment_id = $1",
			evt.Payload.AppointmentID)
		if err != nil {
			log.Printf("[Appointment Consumer] Gagal update status: %v. Requeue.", err)
			d.Nack(false, true)
			continue
		}

		log.Printf("[Appointment Consumer] Stok %s habis (diminta %d, stok %d). Appointment %s -> PENDING_MEDICATION_SUBSTITUTION (%d baris).",
			evt.Payload.DrugCode, evt.Payload.Requested, evt.Payload.CurrentStock, evt.Payload.AppointmentID, tag.RowsAffected())
		d.Ack(false)
	}
}
