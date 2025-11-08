package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateShoppingCartRequest represents the request body for creating a new cart
type CreateShoppingCartRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	FullName   string `json:"full_name" binding:"required"`
}

// CreateShoppingCartResponse represents the response for cart creation
type CreateShoppingCartResponse struct {
	CartID     string    `json:"cart_id"`
	CustomerID string    `json:"customer_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	Message    string    `json:"message"`
}

// ShoppingCartResponse represents full cart with items
type ShoppingCartResponse struct {
	CartID     string           `json:"cart_id"`
	CustomerID string           `json:"customer_id"`
	Status     string           `json:"status"`
	Items      []ShoppingCartItem `json:"items"`
	Total      float64          `json:"total"`
	ItemCount  int              `json:"item_count"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// ShoppingCartItem represents an item in the cart
type ShoppingCartItem struct {
	ItemID       int64   `json:"item_id"`
	ProductID    string  `json:"product_id"`
	ProductName  string  `json:"product_name"`
	Quantity     int     `json:"quantity"`
	PricePerUnit float64 `json:"price_per_unit"`
	Subtotal     float64 `json:"subtotal"`
}

// POST /shopping-carts - Create new shopping cart
func createShoppingCart(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "database_unavailable",
			"message": "Database connection not configured",
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

	// Start transaction for atomic customer + cart creation
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "transaction_failed",
			"message": "Failed to start database transaction",
		})
		return
	}
	defer tx.Rollback()

	// Create or update customer (UPSERT pattern for idempotency)
	_, err = tx.Exec(`
		INSERT INTO customers (customer_id, email, full_name, created_at, updated_at) 
		VALUES (?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE 
			email = VALUES(email), 
			full_name = VALUES(full_name),
			updated_at = NOW()`,
		req.CustomerID, req.Email, req.FullName)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "customer_creation_failed",
			"message": "Failed to create or update customer",
		})
		return
	}

	// Generate unique cart ID
	cartID := uuid.New().String()
	now := time.Now()

	// Create cart
	_, err = tx.Exec(`
		INSERT INTO carts (cart_id, customer_id, status, created_at, updated_at) 
		VALUES (?, ?, 'active', ?, ?)`,
		cartID, req.CustomerID, now, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "cart_creation_failed",
			"message": "Failed to create shopping cart",
		})
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "commit_failed",
			"message": "Failed to commit transaction",
		})
		return
	}

	// Return successful response
	c.JSON(http.StatusCreated, CreateShoppingCartResponse{
		CartID:     cartID,
		CustomerID: req.CustomerID,
		Status:     "active",
		CreatedAt:  now,
		Message:    "Shopping cart created successfully",
	})
}

// GET /shopping-carts/:id - Retrieve cart with all items
func getShoppingCart(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "database_unavailable",
			"message": "Database connection not configured",
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

	// Efficient single query with LEFT JOIN to get cart and all items
	// Uses idx_cart_id index for fast retrieval
	rows, err := db.Query(`
		SELECT 
			c.cart_id, 
			c.customer_id, 
			c.status, 
			c.created_at, 
			c.updated_at,
			ci.item_id, 
			ci.product_id, 
			ci.product_name, 
			ci.quantity, 
			ci.price_per_unit
		FROM carts c
		LEFT JOIN cart_items ci ON c.cart_id = ci.cart_id
		WHERE c.cart_id = ?`, cartID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "query_failed",
			"message": "Failed to retrieve shopping cart",
		})
		return
	}
	defer rows.Close()

	var response *ShoppingCartResponse
	items := []ShoppingCartItem{}
	total := 0.0

	// Process all rows (cart metadata + items)
	for rows.Next() {
		var (
			itemID       sql.NullInt64
			productID    sql.NullString
			productName  sql.NullString
			quantity     sql.NullInt32
			pricePerUnit sql.NullFloat64
		)

		// First row initializes cart metadata
		if response == nil {
			response = &ShoppingCartResponse{}
			err = rows.Scan(
				&response.CartID,
				&response.CustomerID,
				&response.Status,
				&response.CreatedAt,
				&response.UpdatedAt,
				&itemID,
				&productID,
				&productName,
				&quantity,
				&pricePerUnit,
			)
		} else {
			// Subsequent rows: reuse cart data, only scan item columns
			var tempCart ShoppingCartResponse
			err = rows.Scan(
				&tempCart.CartID,
				&tempCart.CustomerID,
				&tempCart.Status,
				&tempCart.CreatedAt,
				&tempCart.UpdatedAt,
				&itemID,
				&productID,
				&productName,
				&quantity,
				&pricePerUnit,
			)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "scan_failed",
				"message": "Failed to parse cart data",
			})
			return
		}

		// Add item to list if exists (LEFT JOIN may return NULL for empty carts)
		if itemID.Valid {
			subtotal := float64(quantity.Int32) * pricePerUnit.Float64
			item := ShoppingCartItem{
				ItemID:       itemID.Int64,
				ProductID:    productID.String,
				ProductName:  productName.String,
				Quantity:     int(quantity.Int32),
				PricePerUnit: pricePerUnit.Float64,
				Subtotal:     subtotal,
			}
			items = append(items, item)
			total += subtotal
		}
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "iteration_failed",
			"message": "Failed to process cart items",
		})
		return
	}

	// Cart not found
	if response == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "cart_not_found",
			"message": fmt.Sprintf("Shopping cart with ID '%s' not found", cartID),
		})
		return
	}

	// Populate items and totals
	response.Items = items
	response.Total = total
	response.ItemCount = len(items)

	c.JSON(http.StatusOK, response)
}

// POST /shopping-carts/:id/items - Add or update item in cart
func addItemToShoppingCart(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "database_unavailable",
			"message": "Database connection not configured",
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

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "transaction_failed",
			"message": "Failed to start transaction",
		})
		return
	}
	defer tx.Rollback()

	// Verify cart exists and is active
	var cartStatus string
	err = tx.QueryRow("SELECT status FROM carts WHERE cart_id = ?", cartID).Scan(&cartStatus)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "cart_not_found",
			"message": fmt.Sprintf("Shopping cart with ID '%s' not found", cartID),
		})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "cart_check_failed",
			"message": "Failed to verify cart",
		})
		return
	}

	// Don't allow adding items to checked out carts
	if cartStatus != "active" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "cart_not_active",
			"message": fmt.Sprintf("Cannot add items to cart with status '%s'", cartStatus),
		})
		return
	}

	// UPSERT: Insert new item or update quantity if product already in cart
	// Uses UNIQUE KEY (cart_id, product_id) to detect duplicates
	result, err := tx.Exec(`
		INSERT INTO cart_items 
			(cart_id, product_id, product_name, quantity, price_per_unit, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE 
			quantity = quantity + VALUES(quantity),
			price_per_unit = VALUES(price_per_unit),
			updated_at = NOW()`,
		cartID, req.ProductID, req.ProductName, req.Quantity, req.PricePerUnit)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "item_add_failed",
			"message": "Failed to add item to cart",
		})
		return
	}

	// Update cart's updated_at timestamp
	_, err = tx.Exec("UPDATE carts SET updated_at = NOW() WHERE cart_id = ?", cartID)
	if err != nil {
		// Non-critical error, log but continue
		fmt.Printf("Warning: failed to update cart timestamp: %v\n", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "commit_failed",
			"message": "Failed to commit transaction",
		})
		return
	}

	itemID, _ := result.LastInsertId()

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Item added to cart successfully",
		"item_id":     itemID,
		"cart_id":     cartID,
		"product_id":  req.ProductID,
		"quantity":    req.Quantity,
		"total_price": float64(req.Quantity) * req.PricePerUnit,
	})
}

