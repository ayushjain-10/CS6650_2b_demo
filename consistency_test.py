#!/usr/bin/env python3
"""
DynamoDB Eventual Consistency Investigation

Tests to observe and measure eventual consistency behavior in DynamoDB.
"""

import requests
import time
import sys
from datetime import datetime
import statistics

class ConsistencyTest:
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip('/')
        self.observations = []
    
    def test_read_after_write_consistency(self, iterations=50):
        """
        Test 1: Create cart then immediately retrieve it
        Measures if write is immediately visible
        """
        print("\n" + "="*70)
        print("TEST 1: Read-After-Write Consistency")
        print("="*70)
        print("Creating cart then immediately reading it back...")
        print()
        
        inconsistencies = 0
        delays = []
        
        for i in range(iterations):
            # Create cart
            create_payload = {
                "customer_id": f"consistency-test-{i}",
                "email": f"test{i}@consistency.com",
                "full_name": f"Test User {i}"
            }
            
            create_response = requests.post(
                f"{self.base_url}/shopping-carts/dynamodb",
                json=create_payload
            )
            
            if create_response.status_code != 201:
                print(f"  ❌ Iteration {i}: Failed to create cart")
                continue
            
            cart_id = create_response.json().get("cart_id")
            create_time = time.time()
            
            # Immediately try to read it back
            max_attempts = 10
            found = False
            read_delay = 0
            
            for attempt in range(max_attempts):
                get_response = requests.get(
                    f"{self.base_url}/shopping-carts/dynamodb/{cart_id}"
                )
                
                if get_response.status_code == 200:
                    read_delay = (time.time() - create_time) * 1000
                    found = True
                    break
                
                time.sleep(0.01)  # Wait 10ms between attempts
            
            if found:
                status = "✅" if read_delay < 100 else "⚠️"
                print(f"  {status} Iteration {i+1}: Found in {read_delay:.1f}ms")
                delays.append(read_delay)
            else:
                print(f"  ❌ Iteration {i+1}: Not found after {max_attempts} attempts")
                inconsistencies += 1
        
        print()
        print("Results:")
        print(f"  Inconsistencies observed: {inconsistencies}/{iterations} ({inconsistencies/iterations*100:.1f}%)")
        
        if delays:
            print(f"  Average read delay: {statistics.mean(delays):.2f}ms")
            print(f"  Min delay: {min(delays):.2f}ms")
            print(f"  Max delay: {max(delays):.2f}ms")
            print(f"  p95 delay: {sorted(delays)[int(len(delays)*0.95)]:.2f}ms")
        
        return inconsistencies, delays
    
    def test_add_item_consistency(self, iterations=30):
        """
        Test 2: Add item then immediately fetch cart
        Measures if item is immediately visible
        """
        print("\n" + "="*70)
        print("TEST 2: Add-Item-Then-Read Consistency")
        print("="*70)
        print("Adding item to cart then immediately reading cart...")
        print()
        
        # Create a test cart first
        create_response = requests.post(
            f"{self.base_url}/shopping-carts/dynamodb",
            json={
                "customer_id": "add-item-consistency-test",
                "email": "additem@test.com",
                "full_name": "Add Item Test"
            }
        )
        
        if create_response.status_code != 201:
            print("❌ Failed to create test cart")
            return 0, []
        
        cart_id = create_response.json().get("cart_id")
        print(f"Test cart created: {cart_id}\n")
        
        inconsistencies = 0
        delays = []
        
        for i in range(iterations):
            # Add item
            item_payload = {
                "product_id": f"consistency-prod-{i}",
                "product_name": f"Consistency Product {i}",
                "quantity": 1,
                "price_per_unit": 99.99
            }
            
            add_response = requests.post(
                f"{self.base_url}/shopping-carts/dynamodb/{cart_id}/items",
                json=item_payload
            )
            
            if add_response.status_code != 201:
                print(f"  ❌ Iteration {i}: Failed to add item")
                continue
            
            add_time = time.time()
            
            # Immediately read cart
            max_attempts = 10
            item_found = False
            
            for attempt in range(max_attempts):
                get_response = requests.get(
                    f"{self.base_url}/shopping-carts/dynamodb/{cart_id}"
                )
                
                if get_response.status_code == 200:
                    cart_data = get_response.json()
                    items = cart_data.get("items", [])
                    
                    # Check if our item is in the cart
                    for item in items:
                        if item.get("product_id") == f"consistency-prod-{i}":
                            read_delay = (time.time() - add_time) * 1000
                            item_found = True
                            delays.append(read_delay)
                            break
                    
                    if item_found:
                        break
                
                time.sleep(0.01)
            
            if item_found:
                status = "✅" if read_delay < 100 else "⚠️"
                print(f"  {status} Iteration {i+1}: Item visible in {read_delay:.1f}ms")
            else:
                print(f"  ❌ Iteration {i+1}: Item not visible after {max_attempts} attempts")
                inconsistencies += 1
        
        print()
        print("Results:")
        print(f"  Inconsistencies: {inconsistencies}/{iterations} ({inconsistencies/iterations*100:.1f}%)")
        
        if delays:
            print(f"  Average visibility delay: {statistics.mean(delays):.2f}ms")
            print(f"  Min delay: {min(delays):.2f}ms")
            print(f"  Max delay: {max(delays):.2f}ms")
        
        return inconsistencies, delays
    
    def test_concurrent_updates(self, num_clients=5, updates_per_client=5):
        """
        Test 3: Multiple clients updating same cart simultaneously
        Observes if updates are lost due to eventual consistency
        """
        print("\n" + "="*70)
        print("TEST 3: Concurrent Updates to Same Cart")
        print("="*70)
        print(f"Simulating {num_clients} clients each adding {updates_per_client} items to same cart...")
        print()
        
        # Create a shared cart
        create_response = requests.post(
            f"{self.base_url}/shopping-carts/dynamodb",
            json={
                "customer_id": "concurrent-test",
                "email": "concurrent@test.com",
                "full_name": "Concurrent Test"
            }
        )
        
        if create_response.status_code != 201:
            print("❌ Failed to create test cart")
            return
        
        cart_id = create_response.json().get("cart_id")
        print(f"Shared cart created: {cart_id}")
        
        expected_items = num_clients * updates_per_client
        
        # Simulate concurrent clients
        import threading
        
        def client_worker(client_id):
            for j in range(updates_per_client):
                item_payload = {
                    "product_id": f"client{client_id}-item{j}",
                    "product_name": f"Client {client_id} Item {j}",
                    "quantity": 1,
                    "price_per_unit": 10.0
                }
                
                requests.post(
                    f"{self.base_url}/shopping-carts/dynamodb/{cart_id}/items",
                    json=item_payload
                )
                time.sleep(0.01)  # Small delay between requests
        
        # Start concurrent clients
        threads = []
        start_time = time.time()
        
        for client_id in range(num_clients):
            thread = threading.Thread(target=client_worker, args=(client_id,))
            thread.start()
            threads.append(thread)
        
        # Wait for all clients to finish
        for thread in threads:
            thread.join()
        
        duration = time.time() - start_time
        print(f"\nAll {num_clients} clients completed in {duration:.2f}s")
        
        # Wait a bit for consistency
        print("Waiting 2 seconds for consistency to propagate...")
        time.sleep(2)
        
        # Read cart and count items
        get_response = requests.get(
            f"{self.base_url}/shopping-carts/dynamodb/{cart_id}"
        )
        
        if get_response.status_code == 200:
            cart_data = get_response.json()
            actual_items = cart_data.get("item_count", 0)
            
            print()
            print("Results:")
            print(f"  Expected items: {expected_items}")
            print(f"  Actual items: {actual_items}")
            print(f"  Lost updates: {expected_items - actual_items}")
            
            if actual_items == expected_items:
                print("  ✅ No lost updates!")
            else:
                print(f"  ❌ {expected_items - actual_items} updates lost ({(expected_items-actual_items)/expected_items*100:.1f}%)")
        else:
            print("❌ Failed to retrieve cart")
    
    def run_all_tests(self):
        """Run all consistency tests"""
        print("\n🔬 DynamoDB Eventual Consistency Investigation")
        print("=" * 70)
        print(f"Base URL: {self.base_url}")
        print(f"Testing DynamoDB consistency behavior...")
        print("=" * 70)
        
        # Test 1: Read-after-write
        inc1, delays1 = self.test_read_after_write_consistency(50)
        
        # Test 2: Add item consistency
        inc2, delays2 = self.test_add_item_consistency(30)
        
        # Test 3: Concurrent updates
        self.test_concurrent_updates(5, 5)
        
        # Overall summary
        print("\n" + "="*70)
        print("🎯 OVERALL CONSISTENCY FINDINGS")
        print("="*70)
        print(f"\nRead-after-write inconsistencies: {inc1}/50 ({inc1/50*100:.1f}%)")
        print(f"Add-item inconsistencies: {inc2}/30 ({inc2/30*100:.1f}%)")
        
        if delays1:
            print(f"\nTypical consistency delay: {statistics.mean(delays1):.2f}ms")
        
        print("\nConclusion:")
        if inc1 + inc2 == 0:
            print("  ✅ DynamoDB showing STRONG consistency behavior")
            print("  (Likely due to single-region deployment and ConsistentRead=true)")
        else:
            print(f"  ⚠️  Observed {inc1 + inc2} consistency delays")
            print("  (This is expected with eventual consistency)")
        
        print("\n" + "="*70)


def main():
    if len(sys.argv) < 2:
        print("Usage: python3 consistency_test.py <base_url>")
        print("Example: python3 consistency_test.py http://CS6650L2-alb-819848504.us-west-2.elb.amazonaws.com")
        sys.exit(1)
    
    base_url = sys.argv[1]
    
    test = ConsistencyTest(base_url)
    test.run_all_tests()


if __name__ == "__main__":
    main()

