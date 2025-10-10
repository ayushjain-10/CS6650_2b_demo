## CS6650 Products API - Deploy and Use

This repo contains a simple Products API (Go + Gin) packaged in Docker and deployed on AWS ECS Fargate via Terraform. Follow the steps below to deploy on any machine and test all API responses.

### Prepare Credentials

Retrieve your temporary credentials for region `us-west-2`.
Enter your configuration when prompted:
```
aws configure
```

And set your session token:
```
aws configure set aws_session_token <YOUR-TEMP-SESSION-TOKEN>
```

### Apply Infrastructure
```
cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/terraform
terraform init -upgrade
terraform apply -auto-approve
```

### Build and Push Docker image to ECR
After Terraform creates the ECR repository, build and push your image locally:
```
# Get ECR repo URL
REPO_URL=$(cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/terraform && terraform output -raw ecr_repository_url)
AWS_REGION=us-west-2

# Login to ECR
aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $REPO_URL

# Build, tag, and push
cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/src
docker build -t app:latest .
docker tag app:latest $REPO_URL:latest
docker push $REPO_URL:latest
```

Force a new ECS deployment to pick up the image:
```
aws ecs update-service \
  --cluster $(cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/terraform && terraform output -raw ecs_cluster_name) \
  --service $(cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/terraform && terraform output -raw ecs_service_name) \
  --force-new-deployment
```

### Send Requests
Make sure you are in terraform folder

Get public ip address
```
aws ec2 describe-network-interfaces \
--network-interface-ids $(
  aws ecs describe-tasks \
    --cluster $(cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/terraform && terraform output -raw ecs_cluster_name) \
    --tasks $(
      aws ecs list-tasks \
        --cluster $(cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/terraform && terraform output -raw ecs_cluster_name) \
        --service-name $(cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/terraform && terraform output -raw ecs_service_name) \
        --query 'taskArns[0]' --output text
    ) \
    --query "tasks[0].attachments[0].details[?name=='networkInterfaceId'].value" \
    --output text
) \
--query 'NetworkInterfaces[0].Association.PublicIp' \
--output text
```

Send some requests (Products API):
```
# List products
curl http://<PUBLIC-IP-ADDRESS>:8080/products

# Create a product
curl -X POST http://<PUBLIC-IP-ADDRESS>:8080/products \
  -H "Content-Type: application/json" \
  -d '{"id":"p1","name":"Widget","description":"A test widget","price":9.99}'

# Get a product by id
curl http://<PUBLIC-IP-ADDRESS>:8080/products/p1

# Common HTTP statuses
# 200 OK (GET existing product)
# 201 Created (POST new product)
# 400 Bad Request (invalid JSON or missing fields)
# 404 Not Found (GET unknown id)
# 409 Conflict (POST duplicate id)
```

### Postman
- Import these endpoints into Postman and save as a collection for reuse.

### Repository Layout
- Server code: `src/main.go`
- Dockerfile: `src/Dockerfile`
- Terraform (infra): `terraform/` (modules in `terraform/modules/`)

### Local Testing
```
cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/src
go test ./...
go build -o server && ./server
```

## Clean Up
```
cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/terraform
terraform destroy -auto-approve
```

### .gitignore and Sensitive Data
Keep large/sensitive files out of git:
- Terraform state and cache: `.terraform/`, `*.tfstate*`
- Local env and variables: `*.tfvars`, `.env`, credentials/keys
- Build artifacts: `server`, `*.log`, `*.out`