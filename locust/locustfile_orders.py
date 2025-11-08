
from locust import FastHttpUser, task, between
import random
import json
import os

# Determine which endpoint to use based on environment variable
USE_ASYNC = os.getenv('ASYNC_MODE', 'false').lower() == 'true'
ENDPOINT = '/orders/async' if USE_ASYNC else '/orders/sync'

class OrderUser(FastHttpUser):

    # Wait time between requests (100-500ms as specified)
    wait_time = between(0.1, 0.5)
    
    def on_start(self):
        self.customer_id = random.randint(1000, 9999)
    
    @task
    def create_order(self):

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
