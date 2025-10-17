# HW6 Screenshots Guide

This document lists the key screenshots you should capture for your report.

## CloudWatch Metrics Screenshots

### 1. ECS Service CPU Utilization
**Path:** AWS Console → ECS → Clusters → CS6650L2-cluster → Services → CS6650L2 → Metrics

**What to show:**
- Time range: Last 1 hour (covering all load tests)
- Metric: CPU Utilization
- Show the progression:
  - Baseline (5 users): ~0.5% CPU
  - Breaking point (20 users): ~3.4% CPU
  - Intense load (40 users): ~6.8% CPU

**Key Data Points to Annotate:**
```
18:27 - Baseline test (5 users) - 0.49% avg
18:30-18:33 - Breaking point test (20 users) - 3.4% avg
18:34-18:39 - Intense load test (40 users) - 6.4-6.8% avg
```

---

### 2. ALB Request Count
**Path:** AWS Console → EC2 → Load Balancers → CS6650L2-alb → Monitoring

**What to show:**
- Metric: Request count
- Show the spike patterns corresponding to load tests
- Peak: 6,070 requests/minute

**Key Data Points:**
```
18:30 - 1,617 requests (baseline)
18:31-18:32 - 3,000+ requests (breaking point)
18:35-18:38 - 6,000+ requests (intense load)
```

---

### 3. ALB Target Response Time
**Path:** AWS Console → EC2 → Load Balancers → CS6650L2-alb → Monitoring

**What to show:**
- Metric: Target Response Time
- Show consistency: ~0.75-0.95ms throughout all tests
- Demonstrates efficient service even under load

---

### 4. Target Group Health
**Path:** AWS Console → EC2 → Target Groups → CS6650L2-tg → Targets

**What to show:**
- Screenshot showing 2 healthy targets
- Their IP addresses
- Health check status

**During resilience test, capture:**
- 1 target "draining"
- 1 target "healthy"
- New target "initial" → "healthy"

---

## ECS Service Screenshots

### 5. Service Overview
**Path:** AWS Console → ECS → Clusters → CS6650L2-cluster → Services → CS6650L2

**What to show:**
- Desired count: 2
- Running count: 2
- Pending count: 0
- Load balancer configuration

---

### 6. Auto Scaling Configuration
**Path:** AWS Console → ECS → Clusters → CS6650L2-cluster → Services → CS6650L2 → Auto Scaling

**What to show:**
- Policy name: CS6650L2-cpu-target
- Target value: 70%
- Min capacity: 2
- Max capacity: 4
- Cooldown periods: 300 seconds

---

### 7. Running Tasks
**Path:** AWS Console → ECS → Clusters → CS6650L2-cluster → Tasks

**What to show:**
- List of 2 running tasks
- Their IDs
- CPU and Memory allocation (256 CPU, 512 MB)
- Status: RUNNING

**During resilience test:**
- Screenshot showing task transitioning to STOPPED
- New task in PENDING/PROVISIONING
- Recovery to 2 RUNNING tasks

---

## Locust Load Test Screenshots

### 8. Locust Web UI - Baseline Test
**If you run with web UI:**
```bash
locust -f locustfile_hw6.py --host http://CS6650L2-alb-67761129.us-west-2.elb.amazonaws.com
# Open http://localhost:8089
# Set: 5 users, 1 spawn rate
```

**What to capture:**
- Statistics tab showing:
  - Total requests
  - Requests/second
  - Average response time
  - Failures (should be 0)

---

### 9. Locust Web UI - Breaking Point Test
**Same as above but with 20 users, 5 spawn rate**

**Show:**
- Higher requests/second
- Response time consistency
- 0 failures

---

### 10. Locust Charts
**Path:** Locust Web UI → Charts tab

**What to show:**
- Total Requests per Second graph
- Response Times (median) graph
- Number of Users graph

**Should demonstrate:**
- Response times remain flat as users increase
- Linear increase in RPS with users
- No spikes or degradation

---

## Terminal Output Screenshots

### 11. Locust Terminal Output - Summary
**Command:**
```bash
locust -f locustfile_hw6.py \
  --host http://CS6650L2-alb-67761129.us-west-2.elb.amazonaws.com \
  -u 40 -r 10 -t 5m --headless --only-summary
```

**Capture the summary table showing:**
```
Type     Name                # reqs   # fails |   Avg   Min   Max   Med | req/s
---------|-------------------|---------|--------|---------------------------|------
GET      search_products     23123    0       |   94    77    300   92  | 77.12
         Aggregated          30098    0       |   94    77    301   92  | 100.38
```

---

### 12. CloudWatch CLI Output - CPU Metrics
**Command:**
```bash
aws cloudwatch get-metric-statistics \
  --namespace AWS/ECS --metric-name CPUUtilization \
  --dimensions Name=ServiceName,Value=CS6650L2 Name=ClusterName,Value=CS6650L2-cluster \
  --start-time $(date -u -v-10M +%Y-%m-%dT%H:%M:%S) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
  --period 60 --statistics Average Maximum
```

**Shows:** CPU utilization progression during tests

---

### 13. Resilience Test - Task Stop Command
**Command:**
```bash
aws ecs stop-task --cluster CS6650L2-cluster --task <TASK_ARN> --reason "Resilience testing"
```

**Capture:**
- Command output showing task being stopped
- "DEACTIVATING" status

---

### 14. Resilience Test - Target Health During Failure
**Command:**
```bash
aws elbv2 describe-target-health --target-group-arn <TG_ARN>
```

**Capture the transition:**
1. Before: 2 healthy
2. During: 1 healthy, 1 draining
3. Recovery: 2 healthy (different IPs)

---

### 15. Resilience Test - Zero Failures
**Terminal output from Locust showing:**
```
Type     Name                # reqs   # fails |   Avg
---------|-------------------|---------|--------|--------
         Aggregated          13531    0(0.00%)|   95ms
```

**Key point:** 0 failures even during instance termination

---

## API Response Screenshots

### 16. Health Check Response
**Command:**
```bash
curl http://CS6650L2-alb-67761129.us-west-2.elb.amazonaws.com/health
```

**Response:**
```json
{"status":"healthy"}
```

---

### 17. Search Response
**Command:**
```bash
curl "http://CS6650L2-alb-67761129.us-west-2.elb.amazonaws.com/products/search?q=Alpha" | jq
```

**Response (formatted):**
```json
{
  "products": [
    {"id":"10","name":"Product Alpha 10","category":"Electronics",...},
    ...10 total results
  ],
  "total_found": 10,
  "search_time": "0.000s",
  "products_checked": 100
}
```

**Highlight:** `products_checked: 100` (critical requirement)

---

## Terraform Configuration Screenshots

### 18. ALB Module Configuration
**File:** `terraform/modules/alb/main.tf`

**Show:**
- Target group with target_type = "ip"
- Health check path = "/health"
- Health check interval = 30s

---

### 19. Auto Scaling Module Configuration
**File:** `terraform/modules/ecs/main.tf`

**Show:**
- `aws_appautoscaling_target` with min=2, max=4
- `aws_appautoscaling_policy` with target_value=70

---

## Architecture Diagram (Optional but Recommended)

Create a simple diagram showing:

```
[Users] → [ALB] → [Target Group] → [ECS Tasks (2-4)]
                                        ↓
                                   [Auto Scaling]
                                        ↓
                                   [CloudWatch Metrics]
```

**Tools you can use:**
- draw.io
- Lucidchart
- AWS Architecture Icons
- Or even PowerPoint/Google Slides

---

## Data Tables for Report

### Performance Summary Table

| Test | Users | Duration | Requests | RPS | Avg Response | 95th % | Failures |
|------|-------|----------|----------|-----|--------------|--------|----------|
| Baseline | 5 | 2m | 1,442 | 12.20 | 96ms | 130ms | 0 |
| Breaking Point | 20 | 3m | 8,997 | 50.04 | 94ms | 120ms | 0 |
| Intense Load | 40 | 5m | 30,098 | 100.38 | 94ms | 120ms | 0 |
| Resilience | 30 | 3m | 13,531 | 75.24 | 95ms | 120ms | 0 |

### CPU Utilization Table

| Test | Instances | Avg CPU per Instance | Peak CPU | Total Capacity Used |
|------|-----------|----------------------|----------|---------------------|
| Baseline | 2 | 0.5% | 0.96% | 1% |
| Breaking Point | 2 | 1.7% | 3.75% | 3.4% |
| Intense Load | 2 | 3.2% | 6.81% | 6.4% |

### Resilience Test Timeline

| Time | Event | Targets | Response Time | Failures |
|------|-------|---------|---------------|----------|
| 00:00 | Test starts | 2 healthy | 94ms | 0 |
| 00:30 | Stop task | 1 healthy, 1 draining | 95ms | 0 |
| 01:30 | New task healthy | 2 healthy (new IPs) | 94ms | 0 |
| 03:00 | Test ends | 2 healthy | 95ms | 0 |

---

## How to Organize Screenshots in Report

### Recommended Structure:

**Part 2: Performance Bottlenecks**
1. Baseline test Locust output
2. Breaking point test Locust output
3. CPU utilization graph (showing low utilization)
4. Single search response showing `products_checked: 100`

**Part 3: Horizontal Scaling**
5. ECS service showing 2 instances
6. Auto Scaling configuration
7. Target group showing 2 healthy targets
8. ALB metrics (request count, response time)
9. Resilience test sequence (3-4 screenshots)
10. Architecture diagram

---

## Key Points to Annotate in Screenshots

1. **CPU Utilization:** Annotate the test periods
2. **Request Count:** Highlight the spike patterns
3. **Target Health:** Circle the "healthy" status
4. **Locust Results:** Highlight "0 failures"
5. **Response Format:** Circle `products_checked: 100`
6. **Resilience Test:** Draw timeline showing recovery

---

## Quick Screenshot Checklist

- [ ] CloudWatch ECS CPU graph
- [ ] CloudWatch ALB request count
- [ ] CloudWatch ALB response time
- [ ] Target group health (2 healthy)
- [ ] Auto scaling configuration
- [ ] Locust baseline results
- [ ] Locust breaking point results
- [ ] Locust intense load results
- [ ] Resilience test (before/during/after)
- [ ] API response with products_checked
- [ ] Terraform configuration files

---

**Total Estimated Screenshots:** 12-15 (more if you include each test variation)

**Report Length:** 1-2 pages as specified, plus appendix with detailed data tables

