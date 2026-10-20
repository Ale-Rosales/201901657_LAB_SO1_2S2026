package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"time"

	pb "munk8s/proto"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedWarReportServiceServer
	ch    *amqp.Channel
	queue string
}

func (s *server) SendReport(ctx context.Context, r *pb.WarReportRequest) (*pb.WarReportResponse, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"country":           r.Country.String(),
		"warplanes_in_air":  r.WarplanesInAir,
		"warships_in_water": r.WarshipsInWater,
		"timestamp":         r.Timestamp,
	})
	err := s.ch.PublishWithContext(ctx, "", s.queue, false, false,
		amqp.Publishing{ContentType: "application/json", Body: body})
	if err != nil {
		return nil, err
	}
	return &pb.WarReportResponse{Status: "ok"}, nil
}

func connect(url string) *amqp.Connection {
	for i := 0; i < 30; i++ {
		c, err := amqp.Dial(url)
		if err == nil {
			return c
		}
		log.Println("esperando rabbitmq:", err)
		time.Sleep(2 * time.Second)
	}
	log.Fatal("no se pudo conectar a rabbitmq")
	return nil
}

func main() {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@rabbitmq:5672/"
	}
	queue := "war_reports"

	conn := connect(url)
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		log.Fatal(err)
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}
	g := grpc.NewServer()
	pb.RegisterWarReportServiceServer(g, &server{ch: ch, queue: queue})
	log.Println("go-server gRPC escuchando en :50051")
	log.Fatal(g.Serve(lis))
}
