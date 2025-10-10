package main

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

type product struct {
	ID          string  `json:"id" binding:"required"`
	Name        string  `json:"name" binding:"required,min=1"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gt=0"`
}

type productStore struct {
	mu       sync.RWMutex
	products map[string]product
}

func newProductStore() *productStore {
	return &productStore{products: make(map[string]product)}
}

func (s *productStore) list() []product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]product, 0, len(s.products))
	for _, p := range s.products {
		items = append(items, p)
	}
	return items
}

func (s *productStore) get(id string) (product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	return p, ok
}

func (s *productStore) create(p product) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.products[p.ID]; exists {
		return errConflict
	}
	s.products[p.ID] = p
	return nil
}

var (
	store       = newProductStore()
	errConflict = gin.Error{Err: nil, Type: gin.ErrorTypePublic, Meta: "conflict"}
)

func main() {
	router := gin.Default()
	router.GET("/products", getProducts)
	router.GET("/products/:id", getProductByID)
	router.POST("/products", postProducts)

	router.Run(":8080")
}

func getProducts(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, store.list())
}

func postProducts(c *gin.Context) {
	var p product
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if p.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	if p.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price must be positive"})
		return
	}
	if err := store.create(p); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "product with this id already exists"})
		return
	}
	c.IndentedJSON(http.StatusCreated, p)
}

func getProductByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	if p, ok := store.get(id); ok {
		c.IndentedJSON(http.StatusOK, p)
		return
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "product not found"})
}
