package main

import (
	"encoding/json"
	"fmt"

	pb "pharmacy-service/pb"

	"google.golang.org/protobuf/proto"
)

func main() {
	// Objek representasi yang sama
	respProto := &pb.CheckDrugResponse{
		DrugCode:     "MED-AMX-500",
		IsAvailable:  true,
		CurrentStock: 500,
		UnitPrice:    3500.0,
		Message:      "Stok obat mencukupi",
	}

	// Serialisasi Protobuf
	protoBytes, _ := proto.Marshal(respProto)

	// Representasi JSON dari objek yang sama
	respJSON := map[string]interface{}{
		"drug_code":     "MED-AMX-500",
		"is_available":  true,
		"current_stock": 500,
		"unit_price":    3500.0,
		"message":       "Stok obat mencukupi",
	}

	// Serialisasi JSON
	jsonBytes, _ := json.Marshal(respJSON)

	// Menghitung persentase pengurangan ukuran
	efficiency := (1.0 - float64(len(protoBytes))/float64(len(jsonBytes))) * 100.0

	fmt.Println("================ ANALISIS WIRE-SIZE PAYLOAD ================")
	fmt.Printf("Ukuran Payload JSON (HTTP/1.1)    : %d Bytes\n", len(jsonBytes))
	fmt.Printf("Ukuran Payload Protobuf (HTTP/2) : %d Bytes\n", len(protoBytes))
	fmt.Printf("Efisiensi Reduksi Ukuran kawat       : %.2f%%\n", efficiency)
	fmt.Println("============================================================")
}