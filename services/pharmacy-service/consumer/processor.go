package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ProcessedEventRecord adalah catatan deduplikasi (idempotent consumer).
type ProcessedEventRecord struct {
	EventID     string    `bson:"event_id"`
	ProcessedAt time.Time `bson:"processed_at"`
	Status      string    `bson:"status"`
}

// processPrescription memproses satu pesan: validasi, deduplikasi, cek stok,
// lalu ACK. Pesan tidak valid ditolak permanen (NACK tanpa requeue) ke DLQ.
func processPrescription(ctx context.Context, mongoClient *mongo.Client, ch *amqp.Channel, d amqp.Delivery) {
	var evt EventEnvelope
	if err := json.Unmarshal(d.Body, &evt); err != nil {
		log.Printf("[Pharmacy Consumer] Format pesan tidak valid, NACK tanpa requeue: %v", err)
		d.Nack(false, false)
		return
	}

	// Poison message: kuantitas tidak valid -> DLQ.
	for _, item := range evt.Payload.Prescription.Items {
		if item.Quantity <= 0 {
			log.Printf("[Pharmacy Consumer] Terdeteksi Poison Message! Kuantitas tidak valid: %d. Mengirim NACK ke DLQ.", item.Quantity)
			d.Nack(false, false)
			return
		}
	}

	db := mongoClient.Database("medcore_pharmacy_db")
	eventColl := db.Collection("processed_events")

	// 1. Deduplikasi berdasarkan event_id.
	var existing ProcessedEventRecord
	if err := eventColl.FindOne(ctx, bson.M{"event_id": evt.EventID}).Decode(&existing); err == nil {
		log.Printf("[Deduplication Guard] Event %s terdeteksi duplikat! Melewati eksekusi bisnis dan langsung mengirim ACK.", evt.EventID)
		d.Ack(false)
		return
	}

	log.Println("---------------------------------------------------------")
	log.Println("[Pharmacy Consumer] Menerima Resep Baru!")
	log.Printf(" Event ID        : %s", evt.EventID)
	log.Printf(" Appointment ID  : %s", evt.Payload.AppointmentID)
	log.Printf(" Prescription ID : %s", evt.Payload.Prescription.PrescriptionID)

	// 2. Logika bisnis: periksa stok; jika kurang, terbitkan event kompensasi.
	for _, item := range evt.Payload.Prescription.Items {
		log.Printf(" -> Item: %s | Qty: %d", item.DrugCode, item.Quantity)

		var drug struct {
			TotalStock int `bson:"total_stock"`
		}
		err := db.Collection("drugs").FindOne(ctx, bson.M{"drug_code": item.DrugCode}).Decode(&drug)
		if err != nil || drug.TotalStock < item.Quantity {
			log.Printf("[Business Logic] Stok %s tidak cukup (stok: %d, diminta: %d). Menerbitkan event kompensasi.", item.DrugCode, drug.TotalStock, item.Quantity)
			if perr := publishStockExhausted(ctx, ch, evt, item.DrugCode, item.Quantity, drug.TotalStock); perr != nil {
				log.Printf("[Error] Gagal menerbitkan event kompensasi: %v. Requeue pesan.", perr)
				d.Nack(false, true)
				return
			}
		} else {
			log.Printf("[Business Logic] Stok %s cukup (stok: %d). Resep disiapkan.", item.DrugCode, drug.TotalStock)
		}
	}

	// 3. Catat event_id sebagai sudah diproses.
	if _, err := eventColl.InsertOne(ctx, ProcessedEventRecord{EventID: evt.EventID, ProcessedAt: time.Now(), Status: "SUCCESS"}); err != nil {
		log.Printf("[Error] Gagal mencatat deduplikasi: %v. Requeue pesan.", err)
		d.Nack(false, true)
		return
	}

	d.Ack(false)
	log.Printf("[Pharmacy Consumer] Event %s berhasil di-ACK dan dicatat.", evt.EventID)
	log.Println("---------------------------------------------------------")
}

// publishStockExhausted menerbitkan event kompensasi pharmacy.stock.exhausted.
func publishStockExhausted(ctx context.Context, ch *amqp.Channel, src EventEnvelope, drugCode string, requested, current int) error {
	eventID := fmt.Sprintf("evt-stock-%s", src.EventID)
	body, err := json.Marshal(map[string]interface{}{
		"event_id":    eventID,
		"event_name":  "StockExhaustedEvent",
		"occurred_at": time.Now().UTC().Format(time.RFC3339),
		"producer":    "pharmacy-service",
		"payload": map[string]interface{}{
			"appointment_id":     src.Payload.AppointmentID,
			"prescription_id":    src.Payload.Prescription.PrescriptionID,
			"drug_code":          drugCode,
			"requested_quantity": requested,
			"current_stock":      current,
		},
	})
	if err != nil {
		return err
	}
	return ch.PublishWithContext(ctx, "medcore.clinical.events", "pharmacy.stock.exhausted", false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		MessageId:    eventID,
		Type:         "StockExhaustedEvent",
		Body:         body,
	})
}
