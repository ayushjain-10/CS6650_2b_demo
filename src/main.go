package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

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

var (
	store = newProductStore()
)

func main() {
	// Generate 100,000 products at startup
	fmt.Println("Generating 100,000 products...")
	start := time.Now()
	store.generateProducts()
	fmt.Printf("Generated %d products in %v\n", 100000, time.Since(start))

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
