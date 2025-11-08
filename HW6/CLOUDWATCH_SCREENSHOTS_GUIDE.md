# CloudWatch Metrics Screenshot Guide for HW8

Quick guide to capture essential CloudWatch metrics after running the performance test.

---

## Prerequisites
- Performance test completed (`mysql_test_results.json` created)
- AWS Console access
- Infrastructure still running (RDS + ECS)

---

## Screenshot 1: RDS CPU Utilization

**Path:** AWS Console → RDS → Databases → `cs6650l2-mysql` → Monitoring

**What to capture:**
- CPU utilization graph during test
- Time range: Last 1 hour
- Look for the spike when you ran the 150-operation test

**Expected values:**
- Baseline: 2-5%
- During test: 12-18%
- Peak: ~28%

**Screenshot should show:** Line graph with visible spike during test period

---

## Screenshot 2: RDS Database Connections

**Path:** Same page as above (RDS → Monitoring tab)

**What to capture:**
- Database connections graph
- Time range: Last 1 hour

**Expected values:**
- Idle: 1-2 connections
- During test: 8-12 connections (stable)
- Max limit: 90

**Screenshot should show:** Stable connection count during test (proving connection pooling works)

---

## Screenshot 3: RDS Read/Write Latency

**Path:** Same page (RDS → Monitoring tab)

**What to capture:**
- Read latency (blue line)
- Write latency (orange line)
- Time range: Last 1 hour

**Expected values:**
- Read latency: 2-5ms
- Write latency: 3-8ms

**Screenshot should show:** Both latencies staying low and stable

---

## Screenshot 4: ECS Service CPU Utilization

**Path:** AWS Console → ECS → Clusters → `CS6650L2-cluster` → Services → `CS6650L2` → Metrics

**What to capture:**
- CPU utilization graph
- Time range: Last 1 hour

**Expected values:**
- Baseline: 2-5%
- During test: 8-15%
- Peak: ~22%

**Screenshot should show:** CPU spike correlating with test time

---

## Screenshot 5: ALB Target Response Time

**Path:** AWS Console → EC2 → Load Balancers → `CS6650L2-alb` → Monitoring

**What to capture:**
- Target response time graph
- Time range: Last 1 hour

**Expected values:**
- Average: 20-30ms
- During test: Should stay under 50ms
- Spike: May reach 40-50ms

**Screenshot should show:** Response times remaining under 50ms requirement

---

## Screenshot 6: ALB Request Count

**Path:** Same page as above (Load Balancer → Monitoring)

**What to capture:**
- Request count graph
- Time range: Last 1 hour

**Expected values:**
- Baseline: 0-5 requests/minute
- During test: 80-120 requests/minute peak
- Total: 150 requests

**Screenshot should show:** Clear spike showing when test was executed

---

## Screenshot 7: ALB HTTP Response Codes

**Path:** Same page as above (Load Balancer → Monitoring)

**What to capture:**
- HTTPCode_Target_2XX_Count (success)
- HTTPCode_Target_5XX_Count (errors)

**Expected values:**
- 2XX: 150 (100% success)
- 5XX: 0 (zero errors)

**Screenshot should show:** All green (no errors)

---

## Quick CLI Alternative (Optional)

If screenshots are difficult, save metrics to files:

```bash
# RDS CPU
aws cloudwatch get-metric-statistics \
  --namespace AWS/RDS \
  --metric-name CPUUtilization \
  --dimensions Name=DBInstanceIdentifier,Value=cs6650l2-mysql \
  --start-time $(date -u -v-1H +%Y-%m-%dT%H:%M:%S) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
  --period 300 \
  --statistics Average Maximum > rds_cpu.json

# Database Connections
aws cloudwatch get-metric-statistics \
  --namespace AWS/RDS \
  --metric-name DatabaseConnections \
  --dimensions Name=DBInstanceIdentifier,Value=cs6650l2-mysql \
  --start-time $(date -u -v-1H +%Y-%m-%dT%H:%M:%S) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
  --period 300 \
  --statistics Average Maximum > db_connections.json

# ECS CPU
aws cloudwatch get-metric-statistics \
  --namespace AWS/ECS \
  --metric-name CPUUtilization \
  --dimensions Name=ServiceName,Value=CS6650L2 Name=ClusterName,Value=CS6650L2-cluster \
  --start-time $(date -u -v-1H +%Y-%m-%dT%H:%M:%S) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
  --period 300 \
  --statistics Average Maximum > ecs_cpu.json
```

---

## Screenshot Naming Convention

Save screenshots as:
```
1_rds_cpu_utilization.png
2_rds_database_connections.png
3_rds_read_write_latency.png
4_ecs_cpu_utilization.png
5_alb_target_response_time.png
6_alb_request_count.png
7_alb_http_codes.png
```

---

## What to Look For (Pass/Fail)

✅ **Good signs:**
- RDS CPU < 70%
- DB connections stable (not climbing)
- Response times < 50ms
- Zero 5XX errors
- No memory leaks (flat memory graph)

❌ **Red flags:**
- RDS CPU > 80%
- DB connections climbing continuously
- Response times > 100ms
- Any 5XX errors
- Memory growing over time

---

## Time Required
- 5-10 minutes to capture all screenshots
- Best done immediately after running performance test

---

**Tip:** Use full-screen browser and hide unnecessary UI elements for cleaner screenshots.

