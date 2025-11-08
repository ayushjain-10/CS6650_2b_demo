package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// createCart creates a new cart for a customer
// POST /carts
func createCart(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	var req struct {
		CustomerID string `json:"customer_id" binding:"required"`
		Email      string `json:"email"`
		FullName   string `json:"full_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Create customer if not exists (UPSERT pattern)
	_, err = tx.Exec(`
		INSERT INTO customers (customer_id, email, full_name) 
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE email = VALUES(email), full_name = VALUES(full_name)`,
		req.CustomerID, req.Email, req.FullName)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create customer"})
		return
	}

	// Generate cart ID
	cartID := uuid.New().String()

	// Create cart
	_, err = tx.Exec(`
		INSERT INTO carts (cart_id, customer_id, status, created_at, updated_at) 
		VALUES (?, ?, 'active', NOW(), NOW())`,
		cartID, req.CustomerID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create cart"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"cart_id":     cartID,
		"customer_id": req.CustomerID,
		"status":      "active",
		"created_at":  time.Now(),
	})
}

// getCart retrieves a cart with all its items
// GET /carts/:cart_id
func getCart(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	cartID := c.Param("cart_id")
	start := time.Now()

	// Query cart with items in a single query (optimized for <50ms)
	rows, err := db.Query(`
		SELECT 
			c.cart_id, c.customer_id, c.status, c.created_at, c.updated_at,
			ci.item_id, ci.product_id, ci.product_name, ci.quantity, ci.price_per_unit
		FROM carts c
		LEFT JOIN cart_items ci ON c.cart_id = ci.cart_id
		WHERE c.cart_id = ?`, cartID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query cart"})
		return
	}
	defer rows.Close()

	var cart *Cart
	items := []CartItem{}
	total := 0.0

	for rows.Next() {
		var (
			itemID       sql.NullInt64
			productID    sql.NullString
			productName  sql.NullString
			quantity     sql.NullInt32
			pricePerUnit sql.NullFloat64
		)

		if cart == nil {
			cart = &Cart{}
			err = rows.Scan(
				&cart.CartID, &cart.CustomerID, &cart.Status, &cart.CreatedAt, &cart.UpdatedAt,
				&itemID, &productID, &productName, &quantity, &pricePerUnit,
			)
		} else {
			var tempCart Cart
			err = rows.Scan(
				&tempCart.CartID, &tempCart.CustomerID, &tempCart.Status, &tempCart.CreatedAt, &tempCart.UpdatedAt,
				&itemID, &productID, &productName, &quantity, &pricePerUnit,
			)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan cart"})
			return
		}

		// Add item if exists (LEFT JOIN might return NULL for empty carts)
		if itemID.Valid {
			item := CartItem{
				ItemID:       itemID.Int64,
				CartID:       cart.CartID,
				ProductID:    productID.String,
				ProductName:  productName.String,
				Quantity:     int(quantity.Int32),
				PricePerUnit: pricePerUnit.Float64,
			}
			items = append(items, item)
			total += float64(item.Quantity) * item.PricePerUnit
		}
	}

	if cart == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cart not found"})
		return
	}

	cart.Items = items
	cart.Total = total

	queryTime := time.Since(start)

	c.JSON(http.StatusOK, gin.H{
		"cart":       cart,
		"query_time": fmt.Sprintf("%dms", queryTime.Milliseconds()),
	})
}

// addItemToCart adds an item to a cart or updates quantity if exists
// POST /carts/:cart_id/items
func addItemToCart(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	cartID := c.Param("cart_id")

	var req AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify cart exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM carts WHERE cart_id = ?)", cartID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "cart not found"})
		return
	}

	// UPSERT: Insert or update quantity if product already in cart
	result, err := db.Exec(`
		INSERT INTO cart_items (cart_id, product_id, product_name, quantity, price_per_unit, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE 
			quantity = quantity + VALUES(quantity),
			updated_at = NOW()`,
		cartID, req.ProductID, req.ProductName, req.Quantity, req.PricePerUnit)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add item"})
		return
	}

	// Update cart's updated_at timestamp
	_, err = db.Exec("UPDATE carts SET updated_at = NOW() WHERE cart_id = ?", cartID)
	if err != nil {
		// Non-critical, just log it
		fmt.Printf("Warning: failed to update cart timestamp: %v\n", err)
	}

	itemID, _ := result.LastInsertId()

	c.JSON(http.StatusCreated, gin.H{
		"message":    "item added to cart",
		"item_id":    itemID,
		"cart_id":    cartID,
		"product_id": req.ProductID,
		"quantity":   req.Quantity,
	})
}

// updateCartItem updates the quantity of an item in the cart
// PUT /carts/:cart_id/items/:item_id
func updateCartItem(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	cartID := c.Param("cart_id")
	itemID := c.Param("item_id")

	var req UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := db.Exec(`
		UPDATE cart_items 
		SET quantity = ?, updated_at = NOW()
		WHERE item_id = ? AND cart_id = ?`,
		req.Quantity, itemID, cartID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update item"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found in cart"})
		return
	}

	// Update cart's updated_at timestamp
	db.Exec("UPDATE carts SET updated_at = NOW() WHERE cart_id = ?", cartID)

	c.JSON(http.StatusOK, gin.H{
		"message":  "item updated",
		"item_id":  itemID,
		"quantity": req.Quantity,
	})
}

// deleteCartItem removes an item from the cart
// DELETE /carts/:cart_id/items/:item_id
func deleteCartItem(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	cartID := c.Param("cart_id")
	itemID := c.Param("item_id")

	result, err := db.Exec(`
		DELETE FROM cart_items 
		WHERE item_id = ? AND cart_id = ?`,
		itemID, cartID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete item"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found in cart"})
		return
	}

	// Update cart's updated_at timestamp
	db.Exec("UPDATE carts SET updated_at = NOW() WHERE cart_id = ?", cartID)

	c.JSON(http.StatusOK, gin.H{
		"message": "item removed from cart",
		"item_id": itemID,
	})
}

// getCustomerCarts retrieves all carts for a customer
// GET /customers/:customer_id/carts
func getCustomerCarts(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	customerID := c.Param("customer_id")
	status := c.DefaultQuery("status", "all") // Filter by status: active, checked_out, abandoned, all

	query := `
		SELECT 
			c.cart_id, c.customer_id, c.status, c.created_at, c.updated_at,
			COALESCE(SUM(ci.quantity * ci.price_per_unit), 0) as total
		FROM carts c
		LEFT JOIN cart_items ci ON c.cart_id = ci.cart_id
		WHERE c.customer_id = ?`

	args := []interface{}{customerID}

	if status != "all" {
		query += " AND c.status = ?"
		args = append(args, status)
	}

	query += " GROUP BY c.cart_id ORDER BY c.created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query carts"})
		return
	}
	defer rows.Close()

	carts := []Cart{}
	for rows.Next() {
		var cart Cart
		err := rows.Scan(
			&cart.CartID, &cart.CustomerID, &cart.Status,
			&cart.CreatedAt, &cart.UpdatedAt, &cart.Total,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan cart"})
			return
		}
		carts = append(carts, cart)
	}

	c.JSON(http.StatusOK, gin.H{
		"customer_id": customerID,
		"carts":       carts,
		"count":       len(carts),
	})
}

// deleteCart deletes a cart and all its items
// DELETE /carts/:cart_id
func deleteCart(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	cartID := c.Param("cart_id")

	// CASCADE DELETE will automatically remove cart_items
	result, err := db.Exec("DELETE FROM carts WHERE cart_id = ?", cartID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete cart"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "cart not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "cart deleted",
		"cart_id": cartID,
	})
}

// checkoutCart marks a cart as checked out
// POST /carts/:cart_id/checkout
func checkoutCart(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	cartID := c.Param("cart_id")

	result, err := db.Exec(`
		UPDATE carts 
		SET status = 'checked_out', updated_at = NOW()
		WHERE cart_id = ? AND status = 'active'`,
		cartID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to checkout cart"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "cart not found or already checked out"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "cart checked out successfully",
		"cart_id": cartID,
		"status":  "checked_out",
	})
}

