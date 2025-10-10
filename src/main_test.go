package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// helper to create a fresh router and store for each test
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	// reset global store to a clean instance
	store = newProductStore()

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/products", getProducts)
	r.GET("/products/:id", getProductByID)
	r.POST("/products", postProducts)
	return r
}

func TestListProducts_Empty(t *testing.T) {
	router := setupTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []product
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("expected empty list, got %v", resp)
	}
}

func TestCreateAndGetProduct(t *testing.T) {
	router := setupTestRouter()

	body := []byte(`{"id":"p1","name":"Widget","description":"A test","price":9.99}`)
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body=%s", w.Code, w.Body.String())
	}

	// GET by id should return 200 with same payload id
	req2 := httptest.NewRequest(http.MethodGet, "/products/p1", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", w2.Code, w2.Body.String())
	}
	var got product
	if err := json.Unmarshal(w2.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if got.ID != "p1" || got.Name != "Widget" {
		t.Fatalf("unexpected product: %+v", got)
	}
}

func TestCreateProduct_Validation(t *testing.T) {
	router := setupTestRouter()

	// missing id
	body := []byte(`{"name":"Widget","price":1}`)
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing id, got %d", w.Code)
	}

	// non-positive price
	body2 := []byte(`{"id":"p2","name":"Widget","price":0}`)
	req2 := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-positive price, got %d", w2.Code)
	}
}

func TestCreateProduct_Conflict(t *testing.T) {
	router := setupTestRouter()

	payload := []byte(`{"id":"dup","name":"One","price":1}`)
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 first create, got %d", w.Code)
	}

	// duplicate
	req2 := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(payload))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate id, got %d", w2.Code)
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	router := setupTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/products/unknown", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
