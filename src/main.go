package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/gin-gonic/gin"
)

type product struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Brand       string `json:"brand"`
}

type searchResponse struct {
	Products        []product `json:"products"`
	TotalFound      int       `json:"total_found"`
	SearchTime      string    `json:"search_time"`
	ProductsChecked int       `json:"products_checked"`
}

// Order structures for HW7 - Synchronous vs Async Processing
type Item struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type Order struct {
	OrderID    string    `json:"order_id"`
	CustomerID int       `json:"customer_id"`
	Status     string    `json:"status"` // pending, processing, completed, failed
	Items      []Item    `json:"items"`
	CreatedAt  time.Time `json:"created_at"`
}

// PaymentProcessor simulates a payment service with limited throughput
type PaymentProcessor struct {
	semaphore chan struct{}
	mu        sync.Mutex
	processed int
	failed    int
}

func newPaymentProcessor(maxConcurrent int) *PaymentProcessor {
	return &PaymentProcessor{
		semaphore: make(chan struct{}, maxConcurrent),
	}
}

func (pp *PaymentProcessor) processPayment(order *Order) error {
	// Try to acquire semaphore (this will block if at capacity)
	pp.semaphore <- struct{}{}
	defer func() { <-pp.semaphore }()

	// Simulate 3-second payment verification delay
	time.Sleep(3 * time.Second)

	pp.mu.Lock()
	pp.processed++
	pp.mu.Unlock()

	return nil
}

func (pp *PaymentProcessor) stats() (int, int) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	return pp.processed, pp.failed
}

// Simple UUID generator without external dependencies
func generateOrderID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("order-%s", hex.EncodeToString(b)[:16])
}

type productStore struct {
	mu       sync.RWMutex
	products []product
}

func newProductStore() *productStore {
	return &productStore{products: make([]product, 0, 100000)}
}

func (s *productStore) generateProducts() {
	s.mu.Lock()
	defer s.mu.Unlock()

	brands := []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Eta", "Theta", "Iota", "Kappa"}
	categories := []string{"Electronics", "Books", "Home", "Clothing", "Sports", "Toys", "Automotive", "Health", "Beauty", "Garden"}
	descriptions := []string{"High quality product", "Premium item", "Best seller", "New arrival", "Limited edition", "Professional grade", "Eco-friendly", "Durable design", "Innovative technology", "Classic style"}

	for i := 1; i <= 100000; i++ {
		brand := brands[i%len(brands)]
		category := categories[i%len(categories)]
		description := descriptions[i%len(descriptions)]

		p := product{
			ID:          strconv.Itoa(i),
			Name:        fmt.Sprintf("Product %s %d", brand, i),
			Category:    category,
			Description: fmt.Sprintf("%s - %s", description, brand),
			Brand:       brand,
		}
		s.products = append(s.products, p)
	}
}

func (s *productStore) search(query string, maxResults int) searchResponse {
	start := time.Now()
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	var results []product
	totalFound := 0
	productsChecked := 0

	// Critical requirement: Check exactly 100 products then stop
	maxCheck := 100
	if maxCheck > len(s.products) {
		maxCheck = len(s.products)
	}

	for i := 0; i < maxCheck; i++ {
		productsChecked++
		p := s.products[i]

		// Search in name and category (case-insensitive)
		if strings.Contains(strings.ToLower(p.Name), query) || strings.Contains(strings.ToLower(p.Category), query) {
			totalFound++
			if len(results) < maxResults {
				results = append(results, p)
			}
		}
	}

	searchTime := time.Since(start)
	return searchResponse{
		Products:        results,
		TotalFound:      totalFound,
		SearchTime:      fmt.Sprintf("%.3fs", searchTime.Seconds()),
		ProductsChecked: productsChecked,
	}
}

// AWS clients
var (
	store            = newProductStore()
	paymentProcessor = newPaymentProcessor(5) // Limit to 5 concurrent payments (simulates 5 orders/sec capacity)
	snsClient        *sns.Client
	sqsClient        *sqs.Client
	snsTopicArn      = "arn:aws:sns:us-west-2:891377339099:order-processing-events"
	sqsQueueURL      = "https://sqs.us-west-2.amazonaws.com/891377339099/order-processing-queue"
)

func initAWS() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
	if err != nil {
		fmt.Printf("Unable to load AWS SDK config: %v\n", err)
		return
	}

	snsClient = sns.NewFromConfig(cfg)
	sqsClient = sqs.NewFromConfig(cfg)
	fmt.Println("AWS SDK initialized successfully")
}

func main() {
	// Initialize AWS SDK
	initAWS()

	// Generate 100,000 products at startup
	fmt.Println("Generating 100,000 products...")
	start := time.Now()
	store.generateProducts()
	fmt.Printf("Generated %d products in %v\n", 100000, time.Since(start))

	// Start order processor worker if in worker mode
	if os.Getenv("WORKER_MODE") == "true" {
		fmt.Println("Starting in WORKER MODE - will process orders from SQS")
		startOrderProcessor() // This will block forever
		return
	}

	router := gin.Default()

	// Health check endpoint for ALB
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Search endpoint
	router.GET("/products/search", searchProducts)

	// Keep existing endpoints for compatibility
	router.GET("/products", getProducts)
	router.GET("/products/:id", getProductByID)
	router.POST("/products", postProducts)

	// HW7: Order processing endpoints
	router.POST("/orders/sync", postOrderSync)
	router.POST("/orders/async", postOrderAsync)
	router.GET("/orders/stats", getOrderStats)

	fmt.Println("Starting server on :8080")
	router.Run(":8080")
}

func searchProducts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	result := store.search(query, 20) // Max 20 results
	c.JSON(http.StatusOK, result)
}

func getProducts(c *gin.Context) {
	// Return first 100 products for compatibility
	store.mu.RLock()
	defer store.mu.RUnlock()

	maxResults := 100
	if len(store.products) < maxResults {
		maxResults = len(store.products)
	}

	c.JSON(http.StatusOK, store.products[:maxResults])
}

func postProducts(c *gin.Context) {
	c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "POST not supported in HW6 implementation"})
}

func getProductByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	store.mu.RLock()
	defer store.mu.RUnlock()

	for _, p := range store.products {
		if p.ID == id {
			c.JSON(http.StatusOK, p)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"message": "product not found"})
}

// HW7: Synchronous order processing
func postOrderSync(c *gin.Context) {
	var order Order

	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate order ID if not provided
	if order.OrderID == "" {
		order.OrderID = generateOrderID()
	}
	order.CreatedAt = time.Now()
	order.Status = "pending"

	// Synchronous payment processing - THIS BLOCKS!
	order.Status = "processing"
	startTime := time.Now()

	if err := paymentProcessor.processPayment(&order); err != nil {
		order.Status = "failed"
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":         "payment processing failed",
			"order_id":      order.OrderID,
			"processing_ms": time.Since(startTime).Milliseconds(),
		})
		return
	}

	order.Status = "completed"
	processingTime := time.Since(startTime)

	c.JSON(http.StatusOK, gin.H{
		"order_id":      order.OrderID,
		"status":        order.Status,
		"customer_id":   order.CustomerID,
		"processing_ms": processingTime.Milliseconds(),
		"message":       "order processed successfully",
	})
}

// Get payment processor statistics
func getOrderStats(c *gin.Context) {
	processed, failed := paymentProcessor.stats()
	c.JSON(http.StatusOK, gin.H{
		"processed":      processed,
		"failed":         failed,
		"max_concurrent": 5,
	})
}

// HW7 Phase 3: Asynchronous order processing
func postOrderAsync(c *gin.Context) {
	var order Order

	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate order ID if not provided
	if order.OrderID == "" {
		order.OrderID = generateOrderID()
	}
	order.CreatedAt = time.Now()
	order.Status = "pending"

	// Publish to SNS immediately - NO BLOCKING!
	orderJSON, err := json.Marshal(order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize order"})
		return
	}

	if snsClient != nil {
		_, err = snsClient.Publish(context.TODO(), &sns.PublishInput{
			TopicArn: aws.String(snsTopicArn),
			Message:  aws.String(string(orderJSON)),
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish order"})
			return
		}
	}

	// Return immediately with 202 Accepted
	c.JSON(http.StatusAccepted, gin.H{
		"order_id":  order.OrderID,
		"status":    "pending",
		"message":   "order received and queued for processing",
		"timestamp": order.CreatedAt,
	})
}

// Background worker that polls SQS and processes orders
func startOrderProcessor() {
	// Get number of worker goroutines from environment (default: 1)
	numWorkers := 1
	if w := os.Getenv("NUM_WORKERS"); w != "" {
		if n, err := strconv.Atoi(w); err == nil && n > 0 {
			numWorkers = n
		}
	}

	fmt.Printf("Order processor started with %d worker goroutines, polling SQS queue...\n", numWorkers)

	// Create a channel for messages
	messagesChan := make(chan types.Message, 100)

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			for message := range messagesChan {
				processOrderMessage(message, workerID)
			}
		}(i + 1)
	}

	// Main polling loop
	for {
		// Long polling - wait up to 20 seconds for messages
		result, err := sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(sqsQueueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     20,
			VisibilityTimeout:   30,
		})

		if err != nil {
			fmt.Printf("Error receiving messages from SQS: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// Send messages to worker pool
		for _, message := range result.Messages {
			messagesChan <- message
		}
	}
}

func processOrderMessage(message types.Message, workerID int) {
	// Parse SNS message wrapper
	var snsMessage struct {
		Message string `json:"Message"`
	}

	if err := json.Unmarshal([]byte(*message.Body), &snsMessage); err != nil {
		fmt.Printf("Error parsing SNS message: %v\n", err)
		return
	}

	// Parse the actual order
	var order Order
	if err := json.Unmarshal([]byte(snsMessage.Message), &order); err != nil {
		fmt.Printf("Error parsing order: %v\n", err)
		return
	}

	fmt.Printf("[Worker %d] Processing order: %s (customer: %d)\n", workerID, order.OrderID, order.CustomerID)

	// Process payment (this takes 3 seconds)
	order.Status = "processing"
	if err := paymentProcessor.processPayment(&order); err != nil {
		order.Status = "failed"
		fmt.Printf("[Worker %d] Payment failed for order %s: %v\n", workerID, order.OrderID, err)
	} else {
		order.Status = "completed"
		fmt.Printf("[Worker %d] Payment completed for order %s\n", workerID, order.OrderID)
	}

	// Delete message from queue after successful processing
	if sqsClient != nil {
		_, err := sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(sqsQueueURL),
			ReceiptHandle: message.ReceiptHandle,
		})

		if err != nil {
			fmt.Printf("Error deleting message from queue: %v\n", err)
		}
	}
}
