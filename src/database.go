package main

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

//go:embed schema.sql
var schemaFS embed.FS

var db *sql.DB

// Database models for shopping cart
type Customer struct {
	CustomerID string    `json:"customer_id"`
	Email      string    `json:"email"`
	FullName   string    `json:"full_name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Cart struct {
	CartID     string    `json:"cart_id"`
	CustomerID string    `json:"customer_id"`
	Status     string    `json:"status"` // active, checked_out, abandoned
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Items      []CartItem `json:"items,omitempty"`
	Total      float64    `json:"total,omitempty"`
}

type CartItem struct {
	ItemID       int64     `json:"item_id"`
	CartID       string    `json:"cart_id"`
	ProductID    string    `json:"product_id"`
	ProductName  string    `json:"product_name"`
	Quantity     int       `json:"quantity"`
	PricePerUnit float64   `json:"price_per_unit"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AddItemRequest struct {
	ProductID    string  `json:"product_id" binding:"required"`
	ProductName  string  `json:"product_name" binding:"required"`
	Quantity     int     `json:"quantity" binding:"required,gt=0"`
	PricePerUnit float64 `json:"price_per_unit" binding:"required,gte=0"`
}

type UpdateItemRequest struct {
	Quantity int `json:"quantity" binding:"required,gt=0"`
}

// InitDB initializes the database connection and runs schema migrations
func InitDB() error {
	// Get connection details from environment variables
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")

	// Skip DB initialization if no host configured
	if dbHost == "" {
		fmt.Println("⚠️  No DB_HOST configured - running without database")
		return nil
	}

	// Default values
	if dbPort == "" {
		dbPort = "3306"
	}
	if dbName == "" {
		dbName = "ecommerce"
	}

	// Build DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection with retries (RDS might not be ready immediately)
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		fmt.Printf("Database not ready, retrying (%d/%d)...\n", i+1, maxRetries)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to database after %d retries: %w", maxRetries, err)
	}

	fmt.Printf("✅ Connected to MySQL database at %s:%s\n", dbHost, dbPort)

	// Run schema migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// runMigrations executes the schema.sql file
func runMigrations() error {
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema.sql: %w", err)
	}

	// Execute schema (CREATE TABLE IF NOT EXISTS ensures idempotency)
	_, err = db.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	fmt.Println("✅ Database schema initialized")
	return nil
}

// CloseDB closes the database connection
func CloseDB() {
	if db != nil {
		db.Close()
	}
}

