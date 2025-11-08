#!/bin/bash

# Manual AWS CLI Deployment Script (No Terraform)
# Deploys ECS, ALB, and connects to existing RDS

set -e
REGION="us-west-2"
SERVICE_NAME="CS6650L2"

echo "🚀 Manual Deployment Starting..."
echo "================================"

# Get VPC and Subnets
echo "📡 Getting VPC and subnet information..."
VPC_ID=$(aws ec2 describe-vpcs --region $REGION --filters "Name=is-default,Values=true" --query 'Vpcs[0].VpcId' --output text)
SUBNETS=$(aws ec2 describe-subnets --region $REGION --filters "Name=vpc-id,Values=$VPC_ID" --query 'Subnets[*].SubnetId' --output text)
SUBNET_ARRAY=($SUBNETS)

echo "✅ VPC: $VPC_ID"
echo "✅ Subnets: ${SUBNET_ARRAY[@]}"

# Create Security Group
echo "🔒 Creating security group..."
SG_ID=$(aws ec2 create-security-group \
    --group-name ${SERVICE_NAME}-sg \
    --description "Security group for CS6650L2" \
    --vpc-id $VPC_ID \
    --region $REGION \
    --query 'GroupId' \
    --output text 2>/dev/null || \
    aws ec2 describe-security-groups \
        --region $REGION \
        --filters "Name=group-name,Values=${SERVICE_NAME}-sg" \
        --query 'SecurityGroups[0].GroupId' \
        --output text)

echo "✅ Security Group: $SG_ID"

# Add ingress rules
echo "🔓 Configuring security group rules..."
aws ec2 authorize-security-group-ingress \
    --group-id $SG_ID \
    --protocol tcp \
    --port 8080 \
    --cidr 0.0.0.0/0 \
    --region $REGION 2>/dev/null || echo "Rule already exists"

aws ec2 authorize-security-group-ingress \
    --group-id $SG_ID \
    --protocol tcp \
    --port 80 \
    --cidr 0.0.0.0/0 \
    --region $REGION 2>/dev/null || echo "Rule already exists"

# Get RDS security group and allow access from ECS
echo "🔗 Configuring RDS access..."
RDS_SG=$(aws rds describe-db-instances \
    --db-instance-identifier cs6650l2-mysql \
    --region $REGION \
    --query 'DBInstances[0].VpcSecurityGroups[0].VpcSecurityGroupId' \
    --output text)

aws ec2 authorize-security-group-ingress \
    --group-id $RDS_SG \
    --protocol tcp \
    --port 3306 \
    --source-group $SG_ID \
    --region $REGION 2>/dev/null || echo "RDS rule already exists"

# Create ALB
echo "⚖️  Creating Application Load Balancer..."
ALB_ARN=$(aws elbv2 create-load-balancer \
    --name ${SERVICE_NAME}-alb \
    --subnets ${SUBNET_ARRAY[@]} \
    --security-groups $SG_ID \
    --region $REGION \
    --query 'LoadBalancers[0].LoadBalancerArn' \
    --output text 2>/dev/null || \
    aws elbv2 describe-load-balancers \
        --region $REGION \
        --names ${SERVICE_NAME}-alb \
        --query 'LoadBalancers[0].LoadBalancerArn' \
        --output text)

ALB_DNS=$(aws elbv2 describe-load-balancers \
    --load-balancer-arns $ALB_ARN \
    --region $REGION \
    --query 'LoadBalancers[0].DNSName' \
    --output text)

echo "✅ ALB: $ALB_DNS"

# Create Target Group
echo "🎯 Creating target group..."
TG_ARN=$(aws elbv2 create-target-group \
    --name ${SERVICE_NAME}-tg \
    --protocol HTTP \
    --port 8080 \
    --vpc-id $VPC_ID \
    --target-type ip \
    --health-check-path /health \
    --health-check-interval-seconds 30 \
    --region $REGION \
    --query 'TargetGroups[0].TargetGroupArn' \
    --output text 2>/dev/null || \
    aws elbv2 describe-target-groups \
        --region $REGION \
        --names ${SERVICE_NAME}-tg \
        --query 'TargetGroups[0].TargetGroupArn' \
        --output text)

echo "✅ Target Group: $TG_ARN"

# Create Listener
echo "👂 Creating ALB listener..."
aws elbv2 create-listener \
    --load-balancer-arn $ALB_ARN \
    --protocol HTTP \
    --port 80 \
    --default-actions Type=forward,TargetGroupArn=$TG_ARN \
    --region $REGION 2>/dev/null || echo "Listener already exists"

# Create CloudWatch Log Group
echo "📝 Creating CloudWatch log group..."
aws logs create-log-group \
    --log-group-name /ecs/${SERVICE_NAME} \
    --region $REGION 2>/dev/null || echo "Log group already exists"

# Get RDS endpoint
echo "🗄️  Getting RDS endpoint..."
RDS_ENDPOINT=$(aws rds describe-db-instances \
    --db-instance-identifier cs6650l2-mysql \
    --region $REGION \
    --query 'DBInstances[0].Endpoint.Address' \
    --output text)

echo "✅ RDS: $RDS_ENDPOINT"

# Create ECS Cluster
echo "🐳 Creating ECS cluster..."
aws ecs create-cluster \
    --cluster-name ${SERVICE_NAME}-cluster \
    --region $REGION 2>/dev/null || echo "Cluster already exists"

# Check if ECR image exists
echo "📦 Checking ECR repository..."
ECR_REPO=$(aws ecr describe-repositories \
    --region $REGION \
    --repository-names ecr_service \
    --query 'repositories[0].repositoryUri' \
    --output text 2>/dev/null || echo "")

if [ -z "$ECR_REPO" ]; then
    echo "❌ ECR repository 'ecr_service' not found. Creating it..."
    aws ecr create-repository \
        --repository-name ecr_service \
        --region $REGION
    ECR_REPO=$(aws ecr describe-repositories \
        --region $REGION \
        --repository-names ecr_service \
        --query 'repositories[0].repositoryUri' \
        --output text)
fi

echo "✅ ECR: $ECR_REPO"

# Register Task Definition
echo "📋 Registering ECS task definition..."
cat > /tmp/task-def.json <<EOF
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
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "DB_HOST",
          "value": "${RDS_ENDPOINT}"
        },
        {
          "name": "DB_PORT",
          "value": "3306"
        },
        {
          "name": "DB_NAME",
          "value": "ecommerce"
        },
        {
          "name": "DB_USER",
          "value": "admin"
        },
        {
          "name": "DB_PASSWORD",
          "value": "MySecurePass123!"
        }
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
    --cli-input-json file:///tmp/task-def.json \
    --region $REGION \
    --query 'taskDefinition.taskDefinitionArn' \
    --output text)

echo "✅ Task Definition: $TASK_DEF_ARN"

# Create ECS Service
echo "🚀 Creating ECS service..."
aws ecs create-service \
    --cluster ${SERVICE_NAME}-cluster \
    --service-name ${SERVICE_NAME} \
    --task-definition $TASK_DEF_ARN \
    --desired-count 2 \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[${SUBNET_ARRAY[0]},${SUBNET_ARRAY[1]}],securityGroups=[$SG_ID],assignPublicIp=ENABLED}" \
    --load-balancers targetGroupArn=$TG_ARN,containerName=${SERVICE_NAME}-container,containerPort=8080 \
    --region $REGION 2>/dev/null || echo "Service already exists, updating..."

# If service exists, update it
aws ecs update-service \
    --cluster ${SERVICE_NAME}-cluster \
    --service ${SERVICE_NAME} \
    --task-definition $TASK_DEF_ARN \
    --desired-count 2 \
    --force-new-deployment \
    --region $REGION 2>/dev/null || true

echo ""
echo "================================"
echo "✅ Deployment Complete!"
echo "================================"
echo ""
echo "ALB DNS: http://$ALB_DNS"
echo "RDS Endpoint: $RDS_ENDPOINT"
echo ""
echo "⏳ Waiting for service to become healthy (this may take 2-3 minutes)..."
echo "   You can check status at: AWS Console → ECS → ${SERVICE_NAME}-cluster → ${SERVICE_NAME}"
echo ""
echo "Next: Run the performance test with:"
echo "  cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo"
echo "  python3 performance_test.py http://$ALB_DNS"
echo ""

