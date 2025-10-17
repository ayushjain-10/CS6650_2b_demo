from locust import FastHttpUser, task, between
import random

class SearchUser(FastHttpUser):
    wait_time = between(0.1, 0.5)  # Minimal wait time for stress testing
    
    # Common search terms that should find matches
    search_terms = [
        "Product", "Alpha", "Beta", "Gamma", "Electronics", "Books", "Home", 
        "Clothing", "Sports", "Toys", "Automotive", "Health", "Beauty", "Garden",
        "1", "2", "3", "10", "100", "1000"  # Common IDs
    ]
    
    @task(10)
    def search_products(self):
        """Main search task - weighted heavily"""
        query = random.choice(self.search_terms)
        self.client.get(f"/products/search?q={query}", name="search_products")
    
    @task(1)
    def health_check(self):
        """Health check endpoint"""
        self.client.get("/health", name="health_check")
    
    @task(1)
    def get_products(self):
        """Get products list"""
        self.client.get("/products", name="get_products")
    
    @task(1)
    def get_product_by_id(self):
        """Get specific product by ID"""
        product_id = random.randint(1, 1000)  # Random ID from 1-1000
        self.client.get(f"/products/{product_id}", name="get_product_by_id")
