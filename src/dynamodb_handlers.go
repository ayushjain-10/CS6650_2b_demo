package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var dynamodbClient *dynamodb.Client
var dynamodbTableName string

// DynamoDB cart structure - single table design with embedded items
type DynamoDBCart struct {
	CartID     string              `dynamodbav:"cart_id"`
	CustomerID string              `dynamodbav:"customer_id"`
	Email      string              `dynamodbav:"email,omitempty"`
	FullName   string              `dynamodbav:"full_name,omitempty"`
	Status     string              `dynamodbav:"status"`
	Items      []DynamoDBCartItem  `dynamodbav:"items"`
	CreatedAt  string              `dynamodbav:"created_at"`
	UpdatedAt  string              `dynamodbav:"updated_at"`
	ExpiresAt  int64               `dynamodbav:"expires_at,omitempty"` // TTL for abandoned carts
}

type DynamoDBCartItem struct {
	ProductID    string  `dynamodbav:"product_id"`
	ProductName  string  `dynamodbav:"product_name"`
	Quantity     int     `dynamodbav:"quantity"`
	PricePerUnit float64 `dynamodbav:"price_per_unit"`
}

// InitDynamoDB initializes the DynamoDB client
func InitDynamoDB(cfg aws.Config) {
	dynamodbTableName = os.Getenv("DYNAMODB_TABLE_NAME")
	if dynamodbTableName == "" {
		fmt.Println("⚠️  No DYNAMODB_TABLE_NAME configured - DynamoDB endpoints will be unavailable")
		return
	}

	dynamodbClient = dynamodb.NewFromConfig(cfg)
	fmt.Printf("✅ DynamoDB client initialized (table: %s)\n", dynamodbTableName)
}

// POST /shopping-carts/dynamodb - Create cart in DynamoDB
func createShoppingCartDynamoDB(c *gin.Context) {
	if dynamodbClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "dynamodb_unavailable",
			"message": "DynamoDB not configured",
		})
		return
	}

	var req CreateShoppingCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Create cart object
	now := time.Now()
	cart := DynamoDBCart{
		CartID:     uuid.New().String(),
		CustomerID: req.CustomerID,
		Email:      req.Email,
		FullName:   req.FullName,
		Status:     "active",
		Items:      []DynamoDBCartItem{}, // Empty cart initially
		CreatedAt:  now.Format(time.RFC3339),
		UpdatedAt:  now.Format(time.RFC3339),
		ExpiresAt:  now.Add(30 * 24 * time.Hour).Unix(), // 30 days TTL
	}

	// Convert to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(cart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "marshal_failed",
			"message": "Failed to marshal cart data",
		})
		return
	}

	// Put item in DynamoDB
	_, err = dynamodbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(dynamodbTableName),
		Item:      item,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "dynamodb_put_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, CreateShoppingCartResponse{
		CartID:     cart.CartID,
		CustomerID: cart.CustomerID,
		Status:     cart.Status,
		CreatedAt:  now,
		Message:    "Shopping cart created successfully (DynamoDB)",
	})
}

// GET /shopping-carts/dynamodb/:id - Retrieve cart from DynamoDB
func getShoppingCartDynamoDB(c *gin.Context) {
	if dynamodbClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "dynamodb_unavailable",
			"message": "DynamoDB not configured",
		})
		return
	}

	cartID := c.Param("id")
	if cartID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_cart_id",
			"message": "Cart ID is required",
		})
		return
	}

	// Get item from DynamoDB
	result, err := dynamodbClient.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(dynamodbTableName),
		Key: map[string]types.AttributeValue{
			"cart_id": &types.AttributeValueMemberS{Value: cartID},
		},
		ConsistentRead: aws.Bool(true), // Strong consistency for cart retrieval
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "dynamodb_get_failed",
			"message": err.Error(),
		})
		return
	}

	// Check if item exists
	if result.Item == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "cart_not_found",
			"message": fmt.Sprintf("Shopping cart with ID '%s' not found", cartID),
		})
		return
	}

	// Unmarshal DynamoDB item
	var cart DynamoDBCart
	err = attributevalue.UnmarshalMap(result.Item, &cart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "unmarshal_failed",
			"message": "Failed to parse cart data",
		})
		return
	}

	// Calculate total
	total := 0.0
	items := []ShoppingCartItem{}
	for _, item := range cart.Items {
		subtotal := float64(item.Quantity) * item.PricePerUnit
		items = append(items, ShoppingCartItem{
			ProductID:    item.ProductID,
			ProductName:  item.ProductName,
			Quantity:     item.Quantity,
			PricePerUnit: item.PricePerUnit,
			Subtotal:     subtotal,
		})
		total += subtotal
	}

	// Parse timestamps
	createdAt, _ := time.Parse(time.RFC3339, cart.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, cart.UpdatedAt)

	// Build response
	response := ShoppingCartResponse{
		CartID:     cart.CartID,
		CustomerID: cart.CustomerID,
		Status:     cart.Status,
		Items:      items,
		Total:      total,
		ItemCount:  len(items),
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// POST /shopping-carts/dynamodb/:id/items - Add item to cart in DynamoDB
func addItemToShoppingCartDynamoDB(c *gin.Context) {
	if dynamodbClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "dynamodb_unavailable",
			"message": "DynamoDB not configured",
		})
		return
	}

	cartID := c.Param("id")
	if cartID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_cart_id",
			"message": "Cart ID is required",
		})
		return
	}

	var req AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// First, check if cart exists and get current items
	getResult, err := dynamodbClient.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(dynamodbTableName),
		Key: map[string]types.AttributeValue{
			"cart_id": &types.AttributeValueMemberS{Value: cartID},
		},
		ConsistentRead: aws.Bool(true),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "dynamodb_get_failed",
			"message": err.Error(),
		})
		return
	}

	if getResult.Item == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "cart_not_found",
			"message": fmt.Sprintf("Shopping cart with ID '%s' not found", cartID),
		})
		return
	}

	var cart DynamoDBCart
	err = attributevalue.UnmarshalMap(getResult.Item, &cart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "unmarshal_failed",
			"message": "Failed to parse cart data",
		})
		return
	}

	// Check if cart is active
	if cart.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "cart_not_active",
			"message": fmt.Sprintf("Cannot add items to cart with status '%s'", cart.Status),
		})
		return
	}

	// Check if product already exists in cart
	productExists := false
	for i, item := range cart.Items {
		if item.ProductID == req.ProductID {
			// Update quantity
			cart.Items[i].Quantity += req.Quantity
			cart.Items[i].PricePerUnit = req.PricePerUnit // Update price
			productExists = true
			break
		}
	}

	// Add new item if doesn't exist
	if !productExists {
		cart.Items = append(cart.Items, DynamoDBCartItem{
			ProductID:    req.ProductID,
			ProductName:  req.ProductName,
			Quantity:     req.Quantity,
			PricePerUnit: req.PricePerUnit,
		})
	}

	// Update timestamp
	cart.UpdatedAt = time.Now().Format(time.RFC3339)

	// Marshal updated cart
	updatedItem, err := attributevalue.MarshalMap(cart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "marshal_failed",
			"message": "Failed to marshal updated cart",
		})
		return
	}

	// Put updated cart back to DynamoDB
	_, err = dynamodbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(dynamodbTableName),
		Item:      updatedItem,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "dynamodb_put_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Item added to cart successfully (DynamoDB)",
		"cart_id":     cartID,
		"product_id":  req.ProductID,
		"quantity":    req.Quantity,
		"total_price": float64(req.Quantity) * req.PricePerUnit,
	})
}

// GET /customers/dynamodb/:customer_id/carts - Get customer's carts from DynamoDB
func getCustomerCartsDynamoDB(c *gin.Context) {
	if dynamodbClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "dynamodb_unavailable",
			"message": "DynamoDB not configured",
		})
		return
	}

	customerID := c.Param("customer_id")

	// Query using CustomerIndex GSI
	result, err := dynamodbClient.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String(dynamodbTableName),
		IndexName:              aws.String("CustomerIndex"),
		KeyConditionExpression: aws.String("customer_id = :customer_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":customer_id": &types.AttributeValueMemberS{Value: customerID},
		},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "dynamodb_query_failed",
			"message": err.Error(),
		})
		return
	}

	// Unmarshal carts
	var carts []DynamoDBCart
	err = attributevalue.UnmarshalListOfMaps(result.Items, &carts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "unmarshal_failed",
			"message": "Failed to parse carts",
		})
		return
	}

	// Build simplified response (without full items)
	cartSummaries := []map[string]interface{}{}
	for _, cart := range carts {
		total := 0.0
		for _, item := range cart.Items {
			total += float64(item.Quantity) * item.PricePerUnit
		}

		cartSummaries = append(cartSummaries, map[string]interface{}{
			"cart_id":    cart.CartID,
			"status":     cart.Status,
			"item_count": len(cart.Items),
			"total":      total,
			"created_at": cart.CreatedAt,
			"updated_at": cart.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"customer_id": customerID,
		"carts":       cartSummaries,
		"count":       len(cartSummaries),
	})
}

