package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Order structure matching the main application
type Item struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type Order struct {
	OrderID    string    `json:"order_id"`
	CustomerID int       `json:"customer_id"`
	Status     string    `json:"status"`
	Items      []Item    `json:"items"`
	CreatedAt  time.Time `json:"created_at"`
}

// PaymentProcessor simulates the same 3-second payment delay as ECS workers
func processPayment(order *Order) error {
	fmt.Printf("Processing payment for order %s (customer: %d)\n", order.OrderID, order.CustomerID)
	
	// Simulate 3-second payment verification delay (same as ECS worker)
	time.Sleep(3 * time.Second)
	
	fmt.Printf("Payment completed for order %s\n", order.OrderID)
	return nil
}

// Handler processes SNS messages
func handler(ctx context.Context, snsEvent events.SNSEvent) error {
	startTime := time.Now()
	
	for _, record := range snsEvent.Records {
		snsRecord := record.SNS
		
		// Parse the order from SNS message
		var order Order
		if err := json.Unmarshal([]byte(snsRecord.Message), &order); err != nil {
			fmt.Printf("Error parsing order: %v\n", err)
			continue
		}
		
		fmt.Printf("Lambda processing order: %s (customer: %d)\n", order.OrderID, order.CustomerID)
		
		// Process payment (this takes 3 seconds)
		order.Status = "processing"
		if err := processPayment(&order); err != nil {
			order.Status = "failed"
			fmt.Printf("Payment failed for order %s: %v\n", order.OrderID, err)
			return err
		}
		
		order.Status = "completed"
		processingTime := time.Since(startTime)
		fmt.Printf("Order %s completed in %v (including cold start)\n", order.OrderID, processingTime)
	}
	
	return nil
}

func main() {
	lambda.Start(handler)
}

