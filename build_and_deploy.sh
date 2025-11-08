#!/bin/bash

# Build and Deploy Updated Docker Image

set -e

ECR_REPO="891377339099.dkr.ecr.us-west-2.amazonaws.com/ecr_service"
REGION="us-west-2"

echo "🔐 Logging into ECR..."
aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $ECR_REPO

echo "🐳 Building Docker image..."
cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/src
docker build -t app:latest .

echo "🏷️  Tagging image..."
docker tag app:latest $ECR_REPO:latest

echo "📤 Pushing to ECR..."
docker push $ECR_REPO:latest

echo "🔄 Forcing ECS service update..."
aws ecs update-service \
  --cluster CS6650L2-cluster \
  --service CS6650L2 \
  --force-new-deployment \
  --region $REGION

echo ""
echo "✅ Image pushed and service updating!"
echo "⏳ Wait 2-3 minutes for new tasks to start"
echo ""
echo "Then run the performance test:"
echo "  cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo"
echo "  python3 performance_test.py http://CS6650L2-alb-819848504.us-west-2.elb.amazonaws.com"

