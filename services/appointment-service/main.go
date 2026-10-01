package main

import (
	"context"
	"log"
	"time"

	pb "appointment-service/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// checkDrug memanggil Pharmacy Service lalu menangani error berdasarkan gRPC status code.
func checkDrug(client pb.PharmacyServiceClient, label, drugCode string, qty int32) {
	log.Printf("---------- %s ----------", label)
	log.Printf("[RPC Call] Validasi obat: %s, jumlah diminta: %d", drugCode, qty)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	resp, err := client.CheckDrugAvailability(ctx, &pb.CheckDrugRequest{
		DrugCode:       drugCode,
		QuantityNeeded: qty,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			log.Printf("[ERROR] Bukan error gRPC: %v", err)
			return
		}

		switch st.Code() {
		case codes.NotFound:
			log.Printf("[PERINGATAN KLINIS][NotFound] Kode obat salah atau tidak ada di inventaris. Periksa kembali input dokter. Detail: %s", st.Message())
		case codes.ResourceExhausted:
			log.Printf("[PERINGATAN KLINIS][ResourceExhausted] Permintaan melebihi kuota farmasi. Kurangi jumlah atau hubungi apoteker. Detail: %s", st.Message())
		case codes.DeadlineExceeded:
			log.Printf("[PERINGATAN KLINIS][DeadlineExceeded] Layanan farmasi terlalu lambat merespons. Coba lagi atau gunakan prosedur manual. Detail: %s", st.Message())
		default:
			log.Printf("[ERROR][%s] %s", st.Code(), st.Message())
		}
		return
	}

	log.Println("================== HASIL RESPON gRPC ==================")
	log.Printf("Kode Obat      : %s", resp.GetDrugCode())
	log.Printf("Tersedia       : %t", resp.GetIsAvailable())
	log.Printf("Stok Aktual    : %d", resp.GetCurrentStock())
	log.Printf("Harga Satuan   : Rp %.2f", resp.GetUnitPrice())
	log.Printf("Catatan Sistem : %s", resp.GetMessage())
	log.Println("========================================================")
}

func main() {
	log.Println("[Appointment Service] Menginisialisasi koneksi gRPC ke Pharmacy Service...")

	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Tidak dapat membentuk koneksi ke Pharmacy Service: %v", err)
	}
	defer conn.Close()

	client := pb.NewPharmacyServiceClient(conn)

	// Skenario 1: obat tersedia (sukses)
	checkDrug(client, "Skenario 1: Obat valid", "MED-AMX-500", 20)

	// Skenario 2: kode obat salah -> codes.NotFound
	checkDrug(client, "Skenario 2: Kode obat salah", "MED-TIDAK-ADA", 10)

	// Skenario 3: jumlah > 5000 -> codes.ResourceExhausted
	checkDrug(client, "Skenario 3: Permintaan melebihi kuota", "MED-AMX-500", 10000)

	// Skenario 4 (tambahan): jumlah melebihi stok tapi <= 5000 -> bukan error, is_available = false
	checkDrug(client, "Skenario 4: Stok tidak cukup", "MED-AMX-500", 600)
}