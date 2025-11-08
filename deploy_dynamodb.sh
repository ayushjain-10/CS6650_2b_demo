#!/bin/bash

# Deploy DynamoDB and Run Performance Test

set -e
REGION="us-west-2"
SERVICE_NAME="CS6650L2"

echo "🚀 DynamoDB Deployment & Testing"
echo "================================="

# Step 1: Create DynamoDB table
echo "📊 Creating DynamoDB table..."
aws dynamodb create-table \
  --table-name ${SERVICE_NAME}-shopping-carts \
  --attribute-definitions \
      AttributeName=cart_id,AttributeType=S \
      AttributeName=customer_id,AttributeType=S \
  --key-schema AttributeName=cart_id,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --global-secondary-indexes \
    "[{
      \"IndexName\": \"CustomerIndex\",
      \"KeySchema\": [{\"AttributeName\":\"customer_id\",\"KeyType\":\"HASH\"}],
      \"Projection\": {\"ProjectionType\":\"ALL\"}
    }]" \
  --region $REGION 2>/dev/null || echo "Table already exists"

# Wait for table to be active
echo "⏳ Waiting for table to be active..."
aws dynamodb wait table-exists \
  --table-name ${SERVICE_NAME}-shopping-carts \
  --region $REGION

# Enable TTL
echo "⏰ Enabling TTL for automatic cart cleanup..."
aws dynamodb update-time-to-live \
  --table-name ${SERVICE_NAME}-shopping-carts \
  --time-to-live-specification "Enabled=true,AttributeName=expires_at" \
  --region $REGION 2>/dev/null || echo "TTL already enabled"

echo "✅ DynamoDB table ready: ${SERVICE_NAME}-shopping-carts"

# Step 2: Download dependencies
echo "📦 Updating Go dependencies..."
cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/src
go mod tidy

# Step 3: Build and push Docker image
echo "🐳 Building Docker image with DynamoDB support..."
ECR_REPO="891377339099.dkr.ecr.us-west-2.amazonaws.com/ecr_service"

aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $ECR_REPO
docker build -t app:latest .
docker tag app:latest $ECR_REPO:latest
docker push $ECR_REPO:latest

# Step 4: Update ECS service with DynamoDB table name
echo "🔄 Updating ECS service..."
TABLE_NAME="${SERVICE_NAME}-shopping-carts"

# Get current task definition
TASK_DEF=$(aws ecs describe-services \
  --cluster ${SERVICE_NAME}-cluster \
  --services ${SERVICE_NAME} \
  --region $REGION \
  --query 'services[0].taskDefinition' \
  --output text)

# Create new task definition with DynamoDB env var
aws ecs describe-task-definition \
  --task-definition $TASK_DEF \
  --region $REGION \
  --query 'taskDefinition' > /tmp/task-def-base.json

# Add DYNAMODB_TABLE_NAME to environment
python3 -c "
import json
with open('/tmp/task-def-base.json', 'r') as f:
    task_def = json.load(f)

# Add DynamoDB table name to environment
for container in task_def['containerDefinitions']:
    if 'environment' not in container:
        container['environment'] = []
    container['environment'].append({
        'name': 'DYNAMODB_TABLE_NAME',
        'value': '${TABLE_NAME}'
    })

# Remove fields that can't be in register call
for field in ['taskDefinitionArn', 'revision', 'status', 'requiresAttributes', 'compatibilities', 'registeredAt', 'registeredBy']:
    task_def.pop(field, None)

with open('/tmp/task-def-new.json', 'w') as f:
    json.dump(task_def, f)
"

# Register new task definition
NEW_TASK_ARN=$(aws ecs register-task-definition \
  --cli-input-json file:///tmp/task-def-new.json \
  --region $REGION \
  --query 'taskDefinition.taskDefinitionArn' \
  --output text)

# Update service
aws ecs update-service \
  --cluster ${SERVICE_NAME}-cluster \
  --service ${SERVICE_NAME} \
  --task-definition $NEW_TASK_ARN \
  --force-new-deployment \
  --region $REGION

echo "✅ ECS service updated with DynamoDB configuration"

# Step 5: Wait for deployment
echo "⏳ Waiting for deployment to complete (2-3 minutes)..."
sleep 120

# Step 6: Get ALB DNS
ALB_DNS=$(aws elbv2 describe-load-balancers \
  --region $REGION \
  --names ${SERVICE_NAME}-alb \
  --query 'LoadBalancers[0].DNSName' \
  --output text)

echo "✅ ALB DNS: http://$ALB_DNS"

# Step 7: Run DynamoDB performance test
echo ""
echo "🧪 Running DynamoDB performance test..."
cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo
python3 performance_test_dynamodb.py http://$ALB_DNS

echo ""
echo "================================="
echo "✅ DynamoDB deployment and testing complete!"
echo "================================="
echo ""
echo "Results:"
echo "  - DynamoDB: dynamodb_test_results.json"
echo "  - MySQL: mysql_test_results.json"
echo ""
echo "Compare results with:"
echo "  python3 -c \"
import json
with open('mysql_test_results.json') as f: mysql = json.load(f)
with open('dynamodb_test_results.json') as f: dynamo = json.load(f)
mysql_avg = sum(r['response_time'] for r in mysql) / len(mysql)
dynamo_avg = sum(r['response_time'] for r in dynamo) / len(dynamo)
print(f'MySQL avg: {mysql_avg:.2f}ms')
print(f'DynamoDB avg: {dynamo_avg:.2f}ms')
print(f'Difference: {dynamo_avg - mysql_avg:+.2f}ms ({(dynamo_avg/mysql_avg-1)*100:+.1f}%)')
  \""

