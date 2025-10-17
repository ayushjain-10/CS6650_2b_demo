## CS6650 Products API - Deploy and Use

This repo contains a simple Products API (Go + Gin) packaged in Docker and deployed on AWS ECS Fargate via Terraform with Application Load Balancer and Auto Scaling. Follow the steps below to deploy on any machine and test all API responses.

**📊 [HW6 Load Testing & Auto-Scaling Report](./HW6/LOAD_TEST_REPORT.md)** - Complete analysis of performance bottlenecks, horizontal scaling, and resilience testing.

### Quick Access

**Load Balancer DNS:** `CS6650L2-alb-67761129.us-west-2.elb.amazonaws.com`

**Live Endpoints:**
- Health: http://CS6650L2-alb-67761129.us-west-2.elb.amazonaws.com/health
- Search: http://CS6650L2-alb-67761129.us-west-2.elb.amazonaws.com/products/search?q=Alpha
- Get All: http://CS6650L2-alb-67761129.us-west-2.elb.amazonaws.com/products
- Get By ID: http://CS6650L2-alb-67761129.us-west-2.elb.amazonaws.com/products/1

---

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

### Load Testing with Locust

The `locust/` folder contains two load testing configurations:

#### **Setup Locust Environment:**
```bash
cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/locust
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

#### **Running Locust Tests:**

**Option 1: Interactive Web UI (Recommended for beginners)**
```bash
# Start Locust web interface
locust -f locustfile_fasthttp.py --host http://$PUBLIC_IP:8080
# Open browser to: http://localhost:8089
# Set users, spawn rate, and start test
```

**Option 2: Headless Mode (Command line)**
```bash
# FastHttpUser (recommended)
locust -f locustfile_fasthttp.py --host http://$PUBLIC_IP:8080 -u 50 -r 10 -t 1m --headless --only-summary

# HttpUser (for comparison)
locust -f locustfile.py --host http://$PUBLIC_IP:8080 -u 50 -r 10 -t 1m --headless --only-summary
```

**Option 3: Custom Test Parameters**
```bash
# High load test
locust -f locustfile_fasthttp.py --host http://$PUBLIC_IP:8080 -u 200 -r 20 -t 5m --headless

# Quick smoke test
locust -f locustfile_fasthttp.py --host http://$PUBLIC_IP:8080 -u 10 -r 2 -t 30s --headless
```

**Parameters Explained:**
- `-u`: Number of users to simulate
- `-r`: Spawn rate (users added per second)
- `-t`: Test duration (e.g., 1m, 30s, 5m)
- `--headless`: Run without web UI
- `--only-summary`: Show only final results

#### **HttpUser vs FastHttpUser Performance Comparison:**
```
Test Configuration: 50 users, 10 spawn rate, 1 minute duration
Target: ECS Fargate service (34.218.41.165:8080)
```

| Metric | HttpUser | FastHttpUser | Improvement |
|--------|----------|--------------|-------------|
| **Total Requests** | 1,673 | 1,823 | **+9.0%** |
| **Requests/sec** | 27.94 | 30.55 | **+9.3%** |
| **Avg Response Time** | 494ms | 332ms | **-32.8%** |
| **GET /products** | 796ms | 490ms | **-38.4%** |
| **POST create** | 209ms | 174ms | **-16.7%** |

**Key Findings:**
- FastHttpUser generates 9% more requests in same time
- 33% faster average response times
- Better resource utilization with asynchronous I/O
- Uses `geventhttpclient` vs synchronous `requests` library

### .gitignore and Sensitive Data
Keep large/sensitive files out of git:
- Terraform state and cache: `.terraform/`, `*.tfstate*`
- Local env and variables: `*.tfvars`, `.env`, credentials/keys
- Build artifacts: `server`, `*.log`, `*.out`