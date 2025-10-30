"""
Locust test file for HW7 - Order Processing Tests
Tests synchronous and asynchronous order processing

Usage:
  Sync Flash Sale (20 users, 60s):
    locust -f locustfile_orders.py --host http://localhost:8080 -u 20 -r 10 -t 60s --headless --only-summary
  
  Async Flash Sale (20 users, 60s) - Set ASYNC_MODE=true:
    ASYNC_MODE=true locust -f locustfile_orders.py --host http://localhost:8080 -u 20 -r 10 -t 60s --headless --only-summary
"""

from locust import FastHttpUser, task, between
import random
import json
import os

# Determine which endpoint to use based on environment variable
USE_ASYNC = os.getenv('ASYNC_MODE', 'false').lower() == 'true'
ENDPOINT = '/orders/async' if USE_ASYNC else '/orders/sync'

class OrderUser(FastHttpUser):
    """
    Simulates users placing orders on the e-commerce platform
    
    Normal operations: 5 concurrent users, 1 user/sec spawn rate
    Flash sale: 20 concurrent users, 10 users/sec spawn rate
    """
    
    # Wait time between requests (100-500ms as specified)
    wait_time = between(0.1, 0.5)
    
    def on_start(self):
        """Called when a simulated user starts"""
        self.customer_id = random.randint(1000, 9999)
    
    @task
    def create_order(self):
        """
        Place an order using either sync or async endpoint
        Sync: blocks for 3 seconds
        Async: returns immediately (<100ms)
        """
        # Generate random order data
        order = {
            "customer_id": self.customer_id,
            "items": [
                {
                    "product_id": f"prod-{random.randint(1, 100)}",
                    "quantity": random.randint(1, 5),
                    "price": round(random.uniform(10.0, 200.0), 2)
                }
            ]
        }
        
        # POST to configured endpoint (sync or async)
        endpoint_name = f"POST {ENDPOINT}"
        self.client.post(ENDPOINT, json=order, name=endpoint_name)
