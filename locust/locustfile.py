from locust import HttpUser, task, between
import random
import string
import json


def random_id(prefix: str = "p") -> str:
    return f"{prefix}-{''.join(random.choices(string.ascii_lowercase + string.digits, k=6))}"


class ProductsUser(HttpUser):
    wait_time = between(0.5, 2.0)

    @task(3)
    def list_products(self):
        self.client.get("/products")

    @task(2)
    def create_product(self):
        pid = random_id()
        payload = {
            "id": pid,
            "name": "Widget",
            "description": "Load test item",
            "price": round(random.uniform(1.0, 50.0), 2),
        }
        headers = {"Content-Type": "application/json"}
        self.client.post("/products", data=json.dumps(payload), headers=headers, name="create_product")

    @task(1)
    def get_unknown(self):
        self.client.get(f"/products/{random_id('unknown')}", name="get_unknown")


