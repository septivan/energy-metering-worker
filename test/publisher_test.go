package anomaly

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type IngestMessage struct {
	RequestID         string    `json:"request_id"`
	ClientFingerprint string    `json:"client_fingerprint"`
	IPAddress         string    `json:"ip_address"`
	UserAgent         string    `json:"user_agent"`
	ReceivedAt        time.Time `json:"received_at"`
	Payload           Payload   `json:"payload"`
}

type Payload struct {
	PM []PMData `json:"PM"`
}

type PMData struct {
	Date string `json:"date"`
	Data string `json:"data"`
	Name string `json:"name"`
}

func main() {
	rabbitURL := flag.String("url", "amqp://guest:guest@localhost:5672/", "RabbitMQ URL")
	exchange := flag.String("exchange", "energy-metering.ingest.exchange", "Exchange name")
	routingKey := flag.String("routing-key", "meter.reading.ingested", "Routing key")
	count := flag.Int("count", 1, "Number of messages to send")
	flag.Parse()

	// Connect to RabbitMQ
	conn, err := amqp.Dial(*rabbitURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

	// Declare exchange
	err = ch.ExchangeDeclare(
		*exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	// Send messages
	for i := 0; i < *count; i++ {
		msg := createTestMessage(i)
		body, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal message %d: %v", i, err)
			continue
		}

		err = ch.Publish(
			*exchange,
			*routingKey,
			false,
			false,
			amqp.Publishing{
				ContentType:  "application/json",
				Body:         body,
				DeliveryMode: amqp.Persistent,
			},
		)
		if err != nil {
			log.Printf("Failed to publish message %d: %v", i, err)
			continue
		}

		log.Printf("Sent message %d: request_id=%s", i+1, msg.RequestID)
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Successfully sent %d messages", *count)
}

func createTestMessage(index int) IngestMessage {
	now := time.Now()

	// Create some variation in the data
	baseValue := 245.5
	variation := float64(index%10) * 5.0

	return IngestMessage{
		RequestID:         uuid.New().String(),
		ClientFingerprint: fmt.Sprintf("test-client-%d", index%3), // 3 different clients
		IPAddress:         "192.168.1.100",
		UserAgent:         "TestPublisher/1.0",
		ReceivedAt:        now,
		Payload: Payload{
			PM: []PMData{
				{
					Date: now.Add(-1 * time.Minute).Format("02/01/2006 15:04:05"),
					Data: fmt.Sprintf("%.2f", baseValue+variation),
					Name: "power_consumption",
				},
				{
					Date: now.Add(-1 * time.Minute).Format("02/01/2006 15:04:05"),
					Data: fmt.Sprintf("%.2f", 220.0+variation),
					Name: "voltage",
				},
				{
					Date: now.Add(-1 * time.Minute).Format("02/01/2006 15:04:05"),
					Data: fmt.Sprintf("%.2f", 50.0),
					Name: "frequency",
				},
			},
		},
	}
}
