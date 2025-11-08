"""
Locust load testing for Shopping Cart API (HW8)

Tests all cart endpoints with realistic shopping scenarios:
- Create cart
- Add items
- Update quantities
- Remove items
- Checkout
- Get customer history

Performance Requirements:
- Cart retrieval: <50ms average
- Support 100 concurrent users
- Handle carts with up to 50 items
"""

from locust import FastHttpUser, task, between
import random
import json

class CartUser(FastHttpUser):
    """
    Simulates a user shopping on the e-commerce site.
    """
    
    wait_time = between(0.5, 2)  # Wait 0.5-2 seconds between requests
    
    def on_start(self):
        """Initialize user session with customer ID and cart."""
        self.customer_id = f"cust-{random.randint(1000, 9999)}"
        self.email = f"{self.customer_id}@example.com"
        self.full_name = f"Test User {random.randint(1, 1000)}"
        self.cart_id = None
        self.items = []  # Track items in cart
        
        # Create a new cart for this user
        self.create_cart()
    
    def create_cart(self):
        """Create a new cart for the customer."""
        response = self.client.post("/carts", 
            json={
                "customer_id": self.customer_id,
                "email": self.email,
                "full_name": self.full_name
            },
            name="POST /carts (create)")
        
        if response.status_code == 201:
            data = response.json()
            self.cart_id = data.get("cart_id")
    
    @task(10)
    def add_item_to_cart(self):
        """Add a random item to the cart (most common operation)."""
        if not self.cart_id:
            return
        
        product_id = f"prod-{random.randint(1, 1000)}"
        product_name = f"Product {random.randint(1, 1000)}"
        quantity = random.randint(1, 5)
        price = round(random.uniform(10.0, 200.0), 2)
        
        response = self.client.post(
            f"/carts/{self.cart_id}/items",
            json={
                "product_id": product_id,
                "product_name": product_name,
                "quantity": quantity,
                "price_per_unit": price
            },
            name="POST /carts/:cart_id/items (add item)"
        )
        
        if response.status_code == 201:
            data = response.json()
            self.items.append({
                "item_id": data.get("item_id"),
                "product_id": product_id,
                "quantity": quantity
            })
    
    @task(15)
    def get_cart(self):
        """Retrieve cart with all items (most frequent read operation)."""
        if not self.cart_id:
            return
        
        response = self.client.get(
            f"/carts/{self.cart_id}",
            name="GET /carts/:cart_id (retrieve)"
        )
        
        if response.status_code == 200:
            data = response.json()
            query_time = data.get("query_time", "N/A")
            # Track query time for performance validation
            # print(f"Cart retrieval time: {query_time}")
    
    @task(3)
    def update_item_quantity(self):
        """Update quantity of an existing item."""
        if not self.cart_id or not self.items:
            return
        
        # Pick a random item from cart
        item = random.choice(self.items)
        new_quantity = random.randint(1, 10)
        
        response = self.client.put(
            f"/carts/{self.cart_id}/items/{item['item_id']}",
            json={"quantity": new_quantity},
            name="PUT /carts/:cart_id/items/:item_id (update)"
        )
        
        if response.status_code == 200:
            item["quantity"] = new_quantity
    
    @task(2)
    def remove_item(self):
        """Remove an item from the cart."""
        if not self.cart_id or not self.items:
            return
        
        # Pick a random item to remove
        item = random.choice(self.items)
        
        response = self.client.delete(
            f"/carts/{self.cart_id}/items/{item['item_id']}",
            name="DELETE /carts/:cart_id/items/:item_id (remove)"
        )
        
        if response.status_code == 200:
            self.items.remove(item)
    
    @task(5)
    def get_customer_carts(self):
        """Get customer's cart history."""
        response = self.client.get(
            f"/customers/{self.customer_id}/carts",
            name="GET /customers/:customer_id/carts (history)"
        )
    
    @task(1)
    def checkout_cart(self):
        """Checkout the cart (least frequent but important)."""
        if not self.cart_id:
            return
        
        response = self.client.post(
            f"/carts/{self.cart_id}/checkout",
            name="POST /carts/:cart_id/checkout"
        )
        
        if response.status_code == 200:
            # After checkout, create a new cart for next shopping session
            self.items = []
            self.create_cart()


class HighLoadCartUser(FastHttpUser):
    """
    Stress test scenario: Users with large carts (up to 50 items).
    Tests the <50ms requirement for carts with many items.
    """
    
    wait_time = between(0.1, 0.5)
    
    def on_start(self):
        """Create a cart with many items."""
        self.customer_id = f"heavy-user-{random.randint(1000, 9999)}"
        self.cart_id = None
        
        # Create cart
        response = self.client.post("/carts", 
            json={
                "customer_id": self.customer_id,
                "email": f"{self.customer_id}@example.com",
                "full_name": "Heavy Cart User"
            })
        
        if response.status_code == 201:
            self.cart_id = response.json().get("cart_id")
            
            # Add 30-50 items to simulate large cart
            num_items = random.randint(30, 50)
            for i in range(num_items):
                self.client.post(
                    f"/carts/{self.cart_id}/items",
                    json={
                        "product_id": f"prod-{i}",
                        "product_name": f"Product {i}",
                        "quantity": random.randint(1, 3),
                        "price_per_unit": round(random.uniform(10.0, 100.0), 2)
                    },
                    name="Setup: Add items"
                )
    
    @task
    def stress_test_large_cart_retrieval(self):
        """Repeatedly retrieve large cart to validate <50ms requirement."""
        if not self.cart_id:
            return
        
        response = self.client.get(
            f"/carts/{self.cart_id}",
            name="GET /carts/:cart_id (large cart)"
        )
        
        if response.status_code == 200:
            data = response.json()
            query_time_str = data.get("query_time", "0ms")
            # Extract milliseconds from "Xms" format
            query_time_ms = int(query_time_str.replace("ms", ""))
            
            # Validate performance requirement
            if query_time_ms > 50:
                print(f"⚠️  SLOW QUERY: {query_time_ms}ms (requirement: <50ms)")

