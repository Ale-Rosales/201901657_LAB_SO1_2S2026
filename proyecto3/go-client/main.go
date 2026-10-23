package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	pb "munk8s/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type report struct {
	Country         string `json:"country"`
	WarplanesInAir  int32  `json:"warplanes_in_air"`
	WarshipsInWater int32  `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

func main() {
	addr := os.Getenv("GRPC_SERVER_ADDR")
	if addr == "" {
		addr = "go-server:50051"
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := pb.NewWarReportServiceClient(conn)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})


	http.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		var in report
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "json invalido", http.StatusBadRequest)
			return
		}
		c, ok := pb.Countries_value[strings.ToLower(in.Country)]
		if !ok || c == 0 {
			http.Error(w, "pais invalido", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		resp, err := client.SendReport(ctx, &pb.WarReportRequest{
			Country:         pb.Countries(c),
			WarplanesInAir:  in.WarplanesInAir,
			WarshipsInWater: in.WarshipsInWater,
			Timestamp:       in.Timestamp,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": resp.Status})
	})

	log.Println("go-client escuchando en :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
