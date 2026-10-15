package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"appointment-service/consumer"
	"appointment-service/publisher"

	"github.com/google/uuid"
)

// ============================================================================
// TUGAS MODUL 3: Asynchronous Messaging & Event Publisher (RabbitMQ)
// ============================================================================

func main() {
	// Flag untuk skenario uji: event_id tetap (uji duplikat), kuantitas obat
	// (uji poison message), dan mode consumer event kompensasi stok habis.
	eventID := flag.String("event-id", "", "event_id manual (kosong = acak)")
	qty := flag.Int("qty", 15, "jumlah obat pada resep (<=0 untuk uji poison message)")
	listen := flag.Bool("listen-compensation", false, "jalankan consumer event stok habis")
	flag.Parse()

	amqpURI := os.Getenv("AMQP_URI")
	if amqpURI == "" {
		amqpURI = "amqp://admin_medcore:SecuredPassRabbit2026!@localhost:5672/medcore_vhost"
	}

	if *listen {
		dsn := os.Getenv("APPOINTMENT_DB_DSN")
		if dsn == "" {
			log.Fatal("APPOINTMENT_DB_DSN belum diset (contoh: postgres://USER:PASS@localhost:5433/medcore_appointment_db)")
		}
		consumer.StartStockExhaustedConsumer(amqpURI, dsn)
		return
	}

	log.Println("[Appointment Service] Menginisialisasi koneksi messaging...")
	pub, err := publisher.NewEventPublisher(amqpURI)
	if err != nil {
		log.Fatalf("Inisialisasi publisher gagal: %v", err)
	}
	defer pub.Close()

	id := *eventID
	if id == "" {
		id = fmt.Sprintf("evt-%s", uuid.New().String())
	}

	// Simulasi event ketika konsultasi selesai
	simulatedEvent := publisher.ConsultationCompletedEvent{
		EventID:    id,
		EventName:  "ConsultationCompletedEvent",
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		Producer:   "appointment-service",
		Payload: publisher.ConsultationPayload{
			AppointmentID:      "b1111111-1111-1111-1111-111111111111",
			PatientID:          "a0000000-0000-0000-0000-000000000001",
			DoctorID:           "b0000000-0000-0000-0000-000000000001",
			ConsultationStatus: "COMPLETED",
			Prescription: publisher.PrescriptionData{
				PrescriptionID: "RX-2026-9901",
				Items: []publisher.PrescriptionItem{
					{
						DrugCode: "MED-AMX-500",
						Quantity: *qty,
						Dosage:   "3x1 sehari sesudah makan",
					},
				},
			},
			BillingSummary: publisher.BillingData{
				ConsultationFee: 150000.0,
				TriageUrgency:   "GREEN",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("[Appointment Service] Menerbitkan event konsultasi ke RabbitMQ...")
	if err := pub.PublishConsultationCompleted(ctx, simulatedEvent); err != nil {
		log.Fatalf("Gagal mempublikasikan event: %v", err)
	}

	log.Println("[Appointment Service] Proses publikasi selesai sempurna.")
}

// ============================================================================
// TUGAS MODUL 2: Synchronous Communication (gRPC Client ke Pharmacy Service)
// (Disimpan sebagai dokumentasi tugas modul 2)
// ============================================================================
//
// import (
// 	pb "appointment-service/pb"
// 	"google.golang.org/grpc"
// 	"google.golang.org/grpc/codes"
// 	"google.golang.org/grpc/credentials/insecure"
// 	"google.golang.org/grpc/status"
// )
//
// // checkDrug memanggil Pharmacy Service lalu menangani error berdasarkan gRPC status code.
// func checkDrug(client pb.PharmacyServiceClient, label, drugCode string, qty int32) {
// 	log.Printf("---------- %s ----------", label)
// 	log.Printf("[RPC Call] Validasi obat: %s, jumlah diminta: %d", drugCode, qty)
//
// 	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
// 	defer cancel()
//
// 	resp, err := client.CheckDrugAvailability(ctx, &pb.CheckDrugRequest{
// 		DrugCode:       drugCode,
// 		QuantityNeeded: qty,
// 	})
// 	if err != nil {
// 		st, ok := status.FromError(err)
// 		if !ok {
// 			log.Printf("[ERROR] Bukan error gRPC: %v", err)
// 			return
// 		}
//
// 		switch st.Code() {
// 		case codes.NotFound:
// 			log.Printf("[PERINGATAN KLINIS][NotFound] Kode obat salah atau tidak ada di inventaris. Periksa kembali input dokter. Detail: %s", st.Message())
// 		case codes.ResourceExhausted:
// 			log.Printf("[PERINGATAN KLINIS][ResourceExhausted] Permintaan melebihi kuota farmasi. Kurangi jumlah atau hubungi apoteker. Detail: %s", st.Message())
// 		case codes.DeadlineExceeded:
// 			log.Printf("[PERINGATAN KLINIS][DeadlineExceeded] Layanan farmasi terlalu lambat merespons. Coba lagi atau gunakan prosedur manual. Detail: %s", st.Message())
// 		default:
// 			log.Printf("[ERROR][%s] %s", st.Code(), st.Message())
// 		}
// 		return
// 	}
//
// 	log.Println("================== HASIL RESPON gRPC ==================")
// 	log.Printf("Kode Obat      : %s", resp.GetDrugCode())
// 	log.Printf("Tersedia       : %t", resp.GetIsAvailable())
// 	log.Printf("Stok Aktual    : %d", resp.GetCurrentStock())
// 	log.Printf("Harga Satuan   : Rp %.2f", resp.GetUnitPrice())
// 	log.Printf("Catatan Sistem : %s", resp.GetMessage())
// 	log.Println("========================================================")
// }
//
// func runModul2() {
// 	log.Println("[Appointment Service] Menginisialisasi koneksi gRPC ke Pharmacy Service...")
//
// 	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
// 	if err != nil {
// 		log.Fatalf("Tidak dapat membentuk koneksi ke Pharmacy Service: %v", err)
// 	}
// 	defer conn.Close()
//
// 	client := pb.NewPharmacyServiceClient(conn)
//
// 	// Skenario 1: obat tersedia (sukses)
// 	checkDrug(client, "Skenario 1: Obat valid", "MED-AMX-500", 20)
//
// 	// Skenario 2: kode obat salah -> codes.NotFound
// 	checkDrug(client, "Skenario 2: Kode obat salah", "MED-TIDAK-ADA", 10)
//
// 	// Skenario 3: jumlah > 5000 -> codes.ResourceExhausted
// 	checkDrug(client, "Skenario 3: Permintaan melebihi kuota", "MED-AMX-500", 10000)
//
// 	// Skenario 4 (tambahan): jumlah melebihi stok tapi <= 5000 -> bukan error, is_available = false
// 	checkDrug(client, "Skenario 4: Stok tidak cukup", "MED-AMX-500", 600)
// }
