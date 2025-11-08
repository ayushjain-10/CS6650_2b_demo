#!/bin/bash

# Complete Deployment and Testing Script for HW8
# This script deploys infrastructure and runs the 150-operation performance test

set -e  # Exit on error

echo "🚀 Starting HW8 Deployment and Testing"
echo "======================================="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Step 1: Deploy Infrastructure
echo -e "${YELLOW}Step 1: Deploying Infrastructure with Terraform${NC}"
cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/terraform

echo "Initializing Terraform..."
terraform init -upgrade

echo "Applying Terraform configuration..."
terraform apply -auto-approve

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Terraform apply failed. Please check errors above.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Infrastructure deployed${NC}"
echo ""

# Step 2: Get ALB DNS
echo -e "${YELLOW}Step 2: Getting ALB DNS name${NC}"
ALB_DNS=$(terraform output -raw alb_dns_name)

if [ -z "$ALB_DNS" ]; then
    echo -e "${RED}❌ Could not get ALB DNS name${NC}"
    exit 1
fi

echo -e "${GREEN}✅ ALB DNS: http://$ALB_DNS${NC}"
echo ""

# Step 3: Wait for service to be healthy
echo -e "${YELLOW}Step 3: Waiting for service to be healthy (may take 2-3 minutes)${NC}"
MAX_ATTEMPTS=30
ATTEMPT=0

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://$ALB_DNS/health 2>/dev/null || echo "000")
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo -e "${GREEN}✅ Service is healthy!${NC}"
        break
    fi
    
    ATTEMPT=$((ATTEMPT + 1))
    echo "Attempt $ATTEMPT/$MAX_ATTEMPTS - Service not ready yet (HTTP $HTTP_CODE), waiting 10 seconds..."
    sleep 10
done

if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
    echo -e "${RED}❌ Service did not become healthy after $MAX_ATTEMPTS attempts${NC}"
    echo "Please check ECS service logs in AWS Console"
    exit 1
fi

echo ""

# Step 4: Quick API test
echo -e "${YELLOW}Step 4: Running quick API test${NC}"

echo "Testing: POST /shopping-carts (create cart)"
CREATE_RESPONSE=$(curl -s -X POST http://$ALB_DNS/shopping-carts \
    -H "Content-Type: application/json" \
    -d '{
        "customer_id": "test-customer-1",
        "email": "test@example.com",
        "full_name": "Test User"
    }')

CART_ID=$(echo $CREATE_RESPONSE | grep -o '"cart_id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$CART_ID" ]; then
    echo -e "${RED}❌ Failed to create cart${NC}"
    echo "Response: $CREATE_RESPONSE"
    exit 1
fi

echo -e "${GREEN}✅ Cart created: $CART_ID${NC}"

echo "Testing: POST /shopping-carts/:id/items (add item)"
curl -s -X POST http://$ALB_DNS/shopping-carts/$CART_ID/items \
    -H "Content-Type: application/json" \
    -d '{
        "product_id": "prod-001",
        "product_name": "Test Product",
        "quantity": 2,
        "price_per_unit": 99.99
    }' > /dev/null

echo -e "${GREEN}✅ Item added to cart${NC}"

echo "Testing: GET /shopping-carts/:id (retrieve cart)"
GET_RESPONSE=$(curl -s http://$ALB_DNS/shopping-carts/$CART_ID)
echo $GET_RESPONSE | grep -q "cart_id"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Cart retrieved successfully${NC}"
else
    echo -e "${RED}❌ Failed to retrieve cart${NC}"
    exit 1
fi

echo ""

# Step 5: Run Performance Test
echo -e "${YELLOW}Step 5: Running 150-Operation Performance Test${NC}"
echo "This will take approximately 2-3 minutes..."
echo ""

cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo

# Check if python3 and requests are available
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}❌ python3 not found${NC}"
    exit 1
fi

# Try to install requests if not present
python3 -c "import requests" 2>/dev/null || {
    echo "Installing requests library..."
    python3 -m pip install requests --quiet
}

# Run the performance test
python3 performance_test.py http://$ALB_DNS

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Performance test failed${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✅ Performance test completed!${NC}"
echo ""

# Step 6: Verify results file
if [ -f "mysql_test_results.json" ]; then
    RESULT_COUNT=$(cat mysql_test_results.json | grep -o '"operation"' | wc -l | tr -d ' ')
    echo -e "${GREEN}✅ Results saved: mysql_test_results.json ($RESULT_COUNT operations)${NC}"
else
    echo -e "${RED}❌ Results file not found${NC}"
    exit 1
fi

echo ""
echo "======================================="
echo -e "${GREEN}🎉 All steps completed successfully!${NC}"
echo "======================================="
echo ""
echo "Next Steps:"
echo "1. Review mysql_test_results.json"
echo "2. Capture CloudWatch screenshots (see HW6/CLOUDWATCH_SCREENSHOTS_GUIDE.md)"
echo ""
echo "Quick screenshot checklist:"
echo "  - AWS Console → RDS → cs6650l2-mysql → Monitoring"
echo "    • CPU Utilization (should show spike)"
echo "    • Database Connections"
echo "  - AWS Console → ECS → CS6650L2-cluster → CS6650L2 → Metrics"
echo "    • CPU Utilization"
echo "  - AWS Console → EC2 → Load Balancers → CS6650L2-alb → Monitoring"
echo "    • Target Response Time"
echo "    • Request Count"
echo ""
echo "Region: us-west-2"
echo "ALB DNS: http://$ALB_DNS"
echo ""

