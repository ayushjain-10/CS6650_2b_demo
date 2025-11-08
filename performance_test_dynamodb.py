#!/usr/bin/env python3
"""
DynamoDB Shopping Cart Performance Test Script

Tests DynamoDB implementation with same 150 operations as MySQL test
for direct comparison.

Usage: python3 performance_test_dynamodb.py <base_url>
"""

import requests
import time
import json
import sys
from datetime import datetime
from typing import List, Dict
import random

class DynamoDBPerformanceTest:
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
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "backend": "dynamodb"
        }
        self.results.append(result)
        return result
    
    def create_cart(self, customer_num: int) -> Dict:
        """POST /shopping-carts/dynamodb - Create cart in DynamoDB"""
        url = f"{self.base_url}/shopping-carts/dynamodb"
        
        payload = {
            "customer_id": f"dynamo-customer-{customer_num}",
            "email": f"dynamo{customer_num}@perftest.com",
            "full_name": f"DynamoDB Test User {customer_num}"
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
        """POST /shopping-carts/dynamodb/{id}/items - Add item to DynamoDB cart"""
        url = f"{self.base_url}/shopping-carts/dynamodb/{cart_id}/items"
        
        payload = {
            "product_id": f"prod-{random.randint(1, 1000)}",
            "product_name": f"DynamoDB Product {item_num}",
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
        """GET /shopping-carts/dynamodb/{id} - Retrieve cart from DynamoDB"""
        url = f"{self.base_url}/shopping-carts/dynamodb/{cart_id}"
        
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
        """Execute the 150-operation test for DynamoDB"""
        print("🚀 Starting DynamoDB Shopping Cart Performance Test")
        print("=" * 60)
        print(f"Base URL: {self.base_url}")
        print(f"Backend: DynamoDB (NoSQL)")
        print(f"Target: 150 operations (50 create + 50 add + 50 get)")
        print("=" * 60)
        print()
        
        test_start = time.time()
        
        # Phase 1: Create 50 carts
        print("Phase 1/3: Creating 50 shopping carts in DynamoDB...")
        for i in range(1, 51):
            result = self.create_cart(i)
            status = "✅" if result["success"] else "❌"
            print(f"{status} Create cart {i}/50: {result['response_time']:.2f}ms (status: {result['status_code']})")
            time.sleep(0.05)
        
        print(f"\n✓ Created {len(self.cart_ids)} carts successfully\n")
        
        if len(self.cart_ids) == 0:
            print("❌ No carts created, cannot continue test")
            return
        
        # Phase 2: Add 50 items
        print("Phase 2/3: Adding 50 items to DynamoDB carts...")
        for i in range(1, 51):
            cart_id = random.choice(self.cart_ids)
            result = self.add_items(cart_id, i)
            status = "✅" if result["success"] else "❌"
            print(f"{status} Add item {i}/50: {result['response_time']:.2f}ms (status: {result['status_code']})")
            time.sleep(0.05)
        
        print(f"\n✓ Added 50 items\n")
        
        # Phase 3: Retrieve 50 carts
        print("Phase 3/3: Retrieving 50 carts from DynamoDB...")
        for i in range(1, 51):
            cart_id = random.choice(self.cart_ids)
            result = self.get_cart(cart_id)
            status = "✅" if result["success"] else "❌"
            print(f"{status} Get cart {i}/50: {result['response_time']:.2f}ms (status: {result['status_code']})")
            time.sleep(0.05)
        
        print(f"\n✓ Retrieved 50 carts\n")
        
        test_duration = time.time() - test_start
        
        # Generate summary
        self.print_summary(test_duration)
        
        # Save results
        self.save_results()
        
        # Compare with MySQL results if available
        self.compare_with_mysql()
    
    def print_summary(self, test_duration: float):
        """Print test summary statistics"""
        print("=" * 60)
        print("📊 DYNAMODB TEST SUMMARY")
        print("=" * 60)
        
        total_ops = len(self.results)
        successful_ops = sum(1 for r in self.results if r["success"])
        failed_ops = total_ops - successful_ops
        
        print(f"Total operations: {total_ops}")
        print(f"Successful: {successful_ops} ({successful_ops/total_ops*100:.1f}%)")
        print(f"Failed: {failed_ops}")
        print(f"Total duration: {test_duration:.2f} seconds")
        print(f"Status: {'✅ PASSED' if test_duration < 300 else '❌ EXCEEDED TIME LIMIT'}")
        print()
        
        # Stats by operation
        for op_type in ["create_cart", "add_items", "get_cart"]:
            op_results = [r for r in self.results if r["operation"] == op_type]
            if op_results:
                response_times = [r["response_time"] for r in op_results]
                successes = sum(1 for r in op_results if r["success"])
                
                print(f"{op_type.upper()}:")
                print(f"  Count: {len(op_results)}")
                print(f"  Success: {successes}/{len(op_results)}")
                print(f"  Avg: {sum(response_times)/len(response_times):.2f}ms")
                print(f"  Min: {min(response_times):.2f}ms")
                print(f"  Max: {max(response_times):.2f}ms")
                
                sorted_times = sorted(response_times)
                p50 = sorted_times[len(sorted_times)//2]
                p95 = sorted_times[int(len(sorted_times)*0.95)]
                p99 = sorted_times[int(len(sorted_times)*0.99)]
                
                print(f"  p50: {p50:.2f}ms")
                print(f"  p95: {p95:.2f}ms")
                print(f"  p99: {p99:.2f}ms")
                print()
        
        print("=" * 60)
    
    def save_results(self, filename: str = "dynamodb_test_results.json"):
        """Save results to JSON file"""
        try:
            with open(filename, 'w') as f:
                json.dump(self.results, f, indent=2)
            print(f"✅ DynamoDB results saved to: {filename}")
        except Exception as e:
            print(f"❌ Failed to save results: {e}")
    
    def compare_with_mysql(self):
        """Compare with MySQL results if available"""
        try:
            with open("mysql_test_results.json", 'r') as f:
                mysql_results = json.load(f)
            
            print("\n" + "=" * 60)
            print("📊 MYSQL vs DYNAMODB COMPARISON")
            print("=" * 60)
            
            # Calculate MySQL stats
            mysql_by_op = {}
            for op in ["create_cart", "add_items", "get_cart"]:
                op_results = [r for r in mysql_results if r["operation"] == op]
                if op_results:
                    times = [r["response_time"] for r in op_results]
                    mysql_by_op[op] = {
                        "avg": sum(times) / len(times),
                        "p95": sorted(times)[int(len(times)*0.95)]
                    }
            
            # Calculate DynamoDB stats
            dynamo_by_op = {}
            for op in ["create_cart", "add_items", "get_cart"]:
                op_results = [r for r in self.results if r["operation"] == op]
                if op_results:
                    times = [r["response_time"] for r in op_results]
                    dynamo_by_op[op] = {
                        "avg": sum(times) / len(times),
                        "p95": sorted(times)[int(len(times)*0.95)]
                    }
            
            # Print comparison
            for op in ["create_cart", "add_items", "get_cart"]:
                if op in mysql_by_op and op in dynamo_by_op:
                    mysql_avg = mysql_by_op[op]["avg"]
                    dynamo_avg = dynamo_by_op[op]["avg"]
                    diff = dynamo_avg - mysql_avg
                    pct_diff = (diff / mysql_avg) * 100
                    
                    print(f"\n{op.upper()}:")
                    print(f"  MySQL avg:    {mysql_avg:.2f}ms")
                    print(f"  DynamoDB avg: {dynamo_avg:.2f}ms")
                    print(f"  Difference:   {diff:+.2f}ms ({pct_diff:+.1f}%)")
            
            print("\n" + "=" * 60)
            
        except FileNotFoundError:
            print("\n⚠️  mysql_test_results.json not found - skipping comparison")


def main():
    if len(sys.argv) < 2:
        print("Usage: python3 performance_test_dynamodb.py <base_url>")
        print("Example: python3 performance_test_dynamodb.py http://CS6650L2-alb-819848504.us-west-2.elb.amazonaws.com")
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
        sys.exit(1)
    
    # Run test
    test = DynamoDBPerformanceTest(base_url)
    test.run_test()
    
    print("\n🎉 DynamoDB performance test completed!")


if __name__ == "__main__":
    main()

