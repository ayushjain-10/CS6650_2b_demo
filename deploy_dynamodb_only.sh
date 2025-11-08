#!/bin/bash

# Deploy ECS with DynamoDB (No RDS)

set -e
REGION="us-west-2"
SERVICE_NAME="CS6650L2"

echo "🚀 Deploying ECS with DynamoDB..."
echo "================================"

# Get infrastructure components
VPC_ID=$(aws ec2 describe-vpcs --region $REGION --filters "Name=is-default,Values=true" --query 'Vpcs[0].VpcId' --output text)
SUBNETS=$(aws ec2 describe-subnets --region $REGION --filters "Name=vpc-id,Values=$VPC_ID" --query 'Subnets[*].SubnetId' --output text)
SUBNET_ARRAY=($SUBNETS)

echo "✅ VPC: $VPC_ID"

# Get or create security group
SG_ID=$(aws ec2 describe-security-groups \
    --region $REGION \
    --filters "Name=group-name,Values=${SERVICE_NAME}-sg" \
    --query 'SecurityGroups[0].GroupId' \
    --output text 2>/dev/null)

if [ "$SG_ID" = "None" ] || [ -z "$SG_ID" ]; then
    SG_ID=$(aws ec2 create-security-group \
        --group-name ${SERVICE_NAME}-sg \
        --description "Security group for CS6650L2" \
        --vpc-id $VPC_ID \
        --region $REGION \
        --query 'GroupId' \
        --output text)
    
    aws ec2 authorize-security-group-ingress \
        --group-id $SG_ID \
        --protocol tcp \
        --port 8080 \
        --cidr 0.0.0.0/0 \
        --region $REGION
    
    aws ec2 authorize-security-group-ingress \
        --group-id $SG_ID \
        --protocol tcp \
        --port 80 \
        --cidr 0.0.0.0/0 \
        --region $REGION
fi

echo "✅ Security Group: $SG_ID"

# Create or get ALB
ALB_ARN=$(aws elbv2 describe-load-balancers \
    --region $REGION \
    --names ${SERVICE_NAME}-alb \
    --query 'LoadBalancers[0].LoadBalancerArn' \
    --output text 2>/dev/null)

if [ "$ALB_ARN" = "None" ] || [ -z "$ALB_ARN" ]; then
    echo "⚖️  Creating ALB..."
    ALB_ARN=$(aws elbv2 create-load-balancer \
        --name ${SERVICE_NAME}-alb \
        --subnets ${SUBNET_ARRAY[@]} \
        --security-groups $SG_ID \
        --region $REGION \
        --query 'LoadBalancers[0].LoadBalancerArn' \
        --output text)
fi

ALB_DNS=$(aws elbv2 describe-load-balancers \
    --load-balancer-arns $ALB_ARN \
    --region $REGION \
    --query 'LoadBalancers[0].DNSName' \
    --output text)

echo "✅ ALB: $ALB_DNS"

# Create or get target group
TG_ARN=$(aws elbv2 describe-target-groups \
    --region $REGION \
    --names ${SERVICE_NAME}-tg \
    --query 'TargetGroups[0].TargetGroupArn' \
    --output text 2>/dev/null)

if [ "$TG_ARN" = "None" ] || [ -z "$TG_ARN" ]; then
    echo "🎯 Creating target group..."
    TG_ARN=$(aws elbv2 create-target-group \
        --name ${SERVICE_NAME}-tg \
        --protocol HTTP \
        --port 8080 \
        --vpc-id $VPC_ID \
        --target-type ip \
        --health-check-path /health \
        --region $REGION \
        --query 'TargetGroups[0].TargetGroupArn' \
        --output text)
    
    # Create listener
    aws elbv2 create-listener \
        --load-balancer-arn $ALB_ARN \
        --protocol HTTP \
        --port 80 \
        --default-actions Type=forward,TargetGroupArn=$TG_ARN \
        --region $REGION
fi

echo "✅ Target Group: $TG_ARN"

# Create cluster if needed
aws ecs create-cluster --cluster-name ${SERVICE_NAME}-cluster --region $REGION 2>/dev/null || true

# Get ECR repo
ECR_REPO=$(aws ecr describe-repositories \
    --region $REGION \
    --repository-names ecr_service \
    --query 'repositories[0].repositoryUri' \
    --output text)

echo "✅ ECR: $ECR_REPO"

# Get DynamoDB table name
DYNAMODB_TABLE="CS6650L2-shopping-carts"
echo "✅ DynamoDB: $DYNAMODB_TABLE"

# Register task definition with DynamoDB env var
echo "📋 Registering task definition..."
cat > /tmp/task-def-dynamodb.json <<EOF
{
  "family": "${SERVICE_NAME}-task",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "executionRoleArn": "arn:aws:iam::891377339099:role/LabRole",
  "taskRoleArn": "arn:aws:iam::891377339099:role/LabRole",
  "containerDefinitions": [
    {
      "name": "${SERVICE_NAME}-container",
      "image": "${ECR_REPO}:latest",
      "essential": true,
      "portMappings": [{"containerPort": 8080}],
      "environment": [
        {"name": "DYNAMODB_TABLE_NAME", "value": "${DYNAMODB_TABLE}"},
        {"name": "DB_HOST", "value": ""},
        {"name": "DB_PORT", "value": "3306"},
        {"name": "DB_NAME", "value": ""},
        {"name": "DB_USER", "value": ""},
        {"name": "DB_PASSWORD", "value": ""}
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/${SERVICE_NAME}",
          "awslogs-region": "${REGION}",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
EOF

TASK_DEF_ARN=$(aws ecs register-task-definition \
    --cli-input-json file:///tmp/task-def-dynamodb.json \
    --region $REGION \
    --query 'taskDefinition.taskDefinitionArn' \
    --output text)

echo "✅ Task Definition: $TASK_DEF_ARN"

# Create ECS service
echo "🚀 Creating ECS service..."
aws ecs create-service \
    --cluster ${SERVICE_NAME}-cluster \
    --service-name ${SERVICE_NAME} \
    --task-definition $TASK_DEF_ARN \
    --desired-count 2 \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[${SUBNET_ARRAY[0]},${SUBNET_ARRAY[1]}],securityGroups=[$SG_ID],assignPublicIp=ENABLED}" \
    --load-balancers targetGroupArn=$TG_ARN,containerName=${SERVICE_NAME}-container,containerPort=8080 \
    --region $REGION 2>&1 | grep -E "(serviceArn|serviceName|status)" | head -5

echo ""
echo "================================"
echo "✅ Deployment Complete!"
echo "================================"
echo ""
echo "ALB DNS: http://$ALB_DNS"
echo "DynamoDB Table: $DYNAMODB_TABLE"
echo ""
echo "⏳ Waiting for service (2 minutes)..."

