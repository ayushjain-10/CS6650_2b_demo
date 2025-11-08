#!/usr/bin/env python3
"""
Shopping Cart API Performance Test Script

Required Test Protocol:
- 50 create cart operations (POST /shopping-carts)
- 50 add items operations (POST /shopping-carts/{id}/items)
- 50 get cart operations (GET /shopping-carts/{id})
- Total: 150 operations
- Complete within 5 minutes
- Save results to: mysql_test_results.json

Output format for each operation:
{
  "operation": "create_cart|add_items|get_cart",
  "response_time": 45.5,  // milliseconds
  "success": true,
  "status_code": 201,
  "timestamp": "2025-01-19T10:00:00Z"
}
"""

import requests
import time
import json
import sys
from datetime import datetime
from typing import List, Dict
import random

class ShoppingCartPerformanceTest:
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip('/')
        self.results: List[Dict] = []
        self.cart_ids: List[str] = []
        
    def record_result(self, operation: str, response_time_ms: float, 
                     success: bool, status_code: int):
        """Record a test result"""
        result = {
            "operation": operation,
            "response_time": round(response_time_ms, 2),
            "success": success,
            "status_code": status_code,
            "timestamp": datetime.utcnow().isoformat() + "Z"
        }
        self.results.append(result)
        return result
    
    def create_cart(self, customer_num: int) -> Dict:
        """POST /shopping-carts - Create new cart"""
        url = f"{self.base_url}/shopping-carts"
        
        payload = {
            "customer_id": f"perf-test-customer-{customer_num}",
            "email": f"customer{customer_num}@perftest.com",
            "full_name": f"Performance Test User {customer_num}"
        }
        
        start_time = time.time()
        try:
            response = requests.post(url, json=payload, timeout=10)
            response_time_ms = (time.time() - start_time) * 1000
            
            success = response.status_code == 201
            result = self.record_result(
                "create_cart",
                response_time_ms,
                success,
                response.status_code
            )
            
            # Store cart ID for subsequent operations
            if success:
                data = response.json()
                cart_id = data.get("cart_id")
                if cart_id:
                    self.cart_ids.append(cart_id)
            
            return result
            
        except Exception as e:
            response_time_ms = (time.time() - start_time) * 1000
            print(f"❌ Create cart failed: {e}")
            return self.record_result("create_cart", response_time_ms, False, 0)
    
    def add_items(self, cart_id: str, item_num: int) -> Dict:
        """POST /shopping-carts/{id}/items - Add item to cart"""
        url = f"{self.base_url}/shopping-carts/{cart_id}/items"
        
        payload = {
            "product_id": f"prod-{random.randint(1, 1000)}",
            "product_name": f"Performance Test Product {item_num}",
            "quantity": random.randint(1, 5),
            "price_per_unit": round(random.uniform(10.0, 200.0), 2)
        }
        
        start_time = time.time()
        try:
            response = requests.post(url, json=payload, timeout=10)
            response_time_ms = (time.time() - start_time) * 1000
            
            success = response.status_code == 201
            return self.record_result(
                "add_items",
                response_time_ms,
                success,
                response.status_code
            )
            
        except Exception as e:
            response_time_ms = (time.time() - start_time) * 1000
            print(f"❌ Add items failed: {e}")
            return self.record_result("add_items", response_time_ms, False, 0)
    
    def get_cart(self, cart_id: str) -> Dict:
        """GET /shopping-carts/{id} - Retrieve cart"""
        url = f"{self.base_url}/shopping-carts/{cart_id}"
        
        start_time = time.time()
        try:
            response = requests.get(url, timeout=10)
            response_time_ms = (time.time() - start_time) * 1000
            
            success = response.status_code == 200
            return self.record_result(
                "get_cart",
                response_time_ms,
                success,
                response.status_code
            )
            
        except Exception as e:
            response_time_ms = (time.time() - start_time) * 1000
            print(f"❌ Get cart failed: {e}")
            return self.record_result("get_cart", response_time_ms, False, 0)
    
    def run_test(self):
        """Execute the complete 150-operation test"""
        print("🚀 Starting Shopping Cart Performance Test")
        print("=" * 60)
        print(f"Base URL: {self.base_url}")
        print(f"Target: 150 operations (50 create + 50 add + 50 get)")
        print(f"Time limit: 5 minutes")
        print("=" * 60)
        print()
        
        test_start = time.time()
        
        # Phase 1: Create 50 carts
        print("Phase 1/3: Creating 50 shopping carts...")
        for i in range(1, 51):
            result = self.create_cart(i)
            status = "✅" if result["success"] else "❌"
            print(f"{status} Create cart {i}/50: {result['response_time']:.2f}ms (status: {result['status_code']})")
            time.sleep(0.05)  # Small delay to avoid overwhelming server
        
        print(f"\n✓ Created {len(self.cart_ids)} carts successfully\n")
        
        if len(self.cart_ids) == 0:
            print("❌ No carts created, cannot continue test")
            return
        
        # Phase 2: Add 50 items (distributed across created carts)
        print("Phase 2/3: Adding 50 items to carts...")
        for i in range(1, 51):
            cart_id = random.choice(self.cart_ids)
            result = self.add_items(cart_id, i)
            status = "✅" if result["success"] else "❌"
            print(f"{status} Add item {i}/50: {result['response_time']:.2f}ms (status: {result['status_code']})")
            time.sleep(0.05)
        
        print(f"\n✓ Added 50 items\n")
        
        # Phase 3: Retrieve 50 carts
        print("Phase 3/3: Retrieving 50 carts...")
        for i in range(1, 51):
            cart_id = random.choice(self.cart_ids)
            result = self.get_cart(cart_id)
            status = "✅" if result["success"] else "❌"
            print(f"{status} Get cart {i}/50: {result['response_time']:.2f}ms (status: {result['status_code']})")
            time.sleep(0.05)
        
        print(f"\n✓ Retrieved 50 carts\n")
        
        test_duration = time.time() - test_start
        
        # Generate summary statistics
        self.print_summary(test_duration)
        
        # Save results to JSON file
        self.save_results()
    
    def print_summary(self, test_duration: float):
        """Print test summary statistics"""
        print("=" * 60)
        print("📊 TEST SUMMARY")
        print("=" * 60)
        
        total_ops = len(self.results)
        successful_ops = sum(1 for r in self.results if r["success"])
        failed_ops = total_ops - successful_ops
        
        print(f"Total operations: {total_ops}")
        print(f"Successful: {successful_ops} ({successful_ops/total_ops*100:.1f}%)")
        print(f"Failed: {failed_ops} ({failed_ops/total_ops*100:.1f}%)")
        print(f"Total duration: {test_duration:.2f} seconds")
        print(f"Time limit: 300 seconds (5 minutes)")
        print(f"Status: {'✅ PASSED' if test_duration < 300 else '❌ EXCEEDED TIME LIMIT'}")
        print()
        
        # Stats by operation type
        for op_type in ["create_cart", "add_items", "get_cart"]:
            op_results = [r for r in self.results if r["operation"] == op_type]
            if op_results:
                response_times = [r["response_time"] for r in op_results]
                successes = sum(1 for r in op_results if r["success"])
                
                print(f"{op_type.upper()}:")
                print(f"  Count: {len(op_results)}")
                print(f"  Success: {successes}/{len(op_results)} ({successes/len(op_results)*100:.1f}%)")
                print(f"  Avg response time: {sum(response_times)/len(response_times):.2f}ms")
                print(f"  Min response time: {min(response_times):.2f}ms")
                print(f"  Max response time: {max(response_times):.2f}ms")
                
                # Calculate percentiles
                sorted_times = sorted(response_times)
                p50 = sorted_times[len(sorted_times)//2]
                p95 = sorted_times[int(len(sorted_times)*0.95)]
                p99 = sorted_times[int(len(sorted_times)*0.99)]
                
                print(f"  p50: {p50:.2f}ms")
                print(f"  p95: {p95:.2f}ms")
                print(f"  p99: {p99:.2f}ms")
                print()
        
        print("=" * 60)
    
    def save_results(self, filename: str = "mysql_test_results.json"):
        """Save results to JSON file"""
        try:
            with open(filename, 'w') as f:
                json.dump(self.results, f, indent=2)
            print(f"✅ Results saved to: {filename}")
            print(f"   Total records: {len(self.results)}")
        except Exception as e:
            print(f"❌ Failed to save results: {e}")


def main():
    if len(sys.argv) < 2:
        print("Usage: python3 performance_test.py <base_url>")
        print("Example: python3 performance_test.py http://CS6650L2-alb-67761129.us-west-2.elb.amazonaws.com")
        sys.exit(1)
    
    base_url = sys.argv[1]
    
    # Verify server is reachable
    try:
        print(f"🔍 Checking server health at {base_url}...")
        response = requests.get(f"{base_url}/health", timeout=5)
        if response.status_code == 200:
            print("✅ Server is healthy\n")
        else:
            print(f"⚠️  Server returned status {response.status_code}\n")
    except Exception as e:
        print(f"❌ Cannot reach server: {e}")
        print("Please verify the URL and ensure the server is running.")
        sys.exit(1)
    
    # Run performance test
    test = ShoppingCartPerformanceTest(base_url)
    test.run_test()
    
    print("\n🎉 Performance test completed!")
    print(f"Review results in: mysql_test_results.json")


if __name__ == "__main__":
    main()

