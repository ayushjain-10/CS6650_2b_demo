# DynamoDB CloudWatch Monitoring - Screenshot Guide

Quick guide to capture DynamoDB metrics after running the 150-operation test.

---

## Where to Find DynamoDB Metrics

**AWS Console → DynamoDB → Tables → CS6650L2-shopping-carts → Monitor tab**

Make sure you're in **us-west-2** region!

---

## Screenshot 1: Request Latency

**Location:** DynamoDB → CS6650L2-shopping-carts → Monitor → Metrics

**Metric to capture:** 
- **SuccessfulRequestLatency** (line graph)

**What to look for:**
- GetItem latency: Should show ~10-30ms
- PutItem latency: Should show ~15-40ms
- Query latency (CustomerIndex): Should show ~20-50ms

**Time range:** Last 1 hour

**Expected values from test:**
- Average latency: 10-30ms
- Spikes during test period

---

## Screenshot 2: Consumed Capacity (Read/Write)

**Location:** Same page (Monitor tab)

**Metrics to capture:**
- **ConsumedReadCapacityUnits** (shows read usage)
- **ConsumedWriteCapacityUnits** (shows write usage)

**What to look for:**
- Read spikes: During GET operations (Phase 3 of test)
- Write spikes: During POST operations (Phase 1 & 2 of test)
- Total consumed: ~200 RCUs + 100 WCUs for 150 operations

**Time range:** Last 1 hour

**Expected pattern:**
```
Phase 1 (create): High writes
Phase 2 (add items): High reads + writes (read-modify-write)
Phase 3 (get): High reads
```

---

## Screenshot 3: Throttling Events

**Location:** Same page (Monitor tab)

**Metrics to capture:**
- **UserErrors** (should be 0)
- **SystemErrors** (should be 0)
- **ThrottledRequests** (should be 0 with on-demand billing)

**What to look for:**
- All should be ZERO (on-demand mode doesn't throttle)
- Any spikes indicate capacity issues

**Expected:** Flat line at 0 (no throttling with PAY_PER_REQUEST)

---

## Screenshot 4: Operation Breakdown

**Location:** Same page (Monitor tab)

**Metrics to capture:**
- **Returned Item Count by Operation** (shows GetItem, PutItem, Query counts)

**What to look for:**
- GetItem: 50 operations (Phase 3)
- PutItem: 100 operations (50 create + 50 add items)
- Query: 0-5 operations (if testing customer history)

**Expected total operations:** ~150

---

## Screenshot 5: Table-Level Metrics

**Location:** DynamoDB → CS6650L2-shopping-carts → Additional settings → Capacity tab

**Metrics to capture:**
- **Read capacity mode:** PAY_PER_REQUEST
- **Write capacity mode:** PAY_PER_REQUEST

**Shows:** On-demand billing enabled (no provisioned capacity)

---

## Screenshot 6: Global Secondary Index Metrics

**Location:** DynamoDB → CS6650L2-shopping-carts → Indexes tab → CustomerIndex

**Metrics to capture (if you queried by customer):**
- CustomerIndex consumed read/write capacity
- CustomerIndex item count

**Expected:** Lower usage (only if you tested customer history endpoint)

---

## Quick CLI Alternative

Save metrics to JSON files:

```bash
REGION="us-west-2"
TABLE_NAME="CS6650L2-shopping-carts"
START_TIME=$(date -u -v-1H +%Y-%m-%dT%H:%M:%S)
END_TIME=$(date -u +%Y-%m-%dT%H:%M:%S)

# Get Item Latency
aws cloudwatch get-metric-statistics \
  --namespace AWS/DynamoDB \
  --metric-name SuccessfulRequestLatency \
  --dimensions Name=TableName,Value=$TABLE_NAME Name=Operation,Value=GetItem \
  --start-time $START_TIME \
  --end-time $END_TIME \
  --period 300 \
  --statistics Average Maximum \
  --region $REGION > dynamodb_getitem_latency.json

# Put Item Latency
aws cloudwatch get-metric-statistics \
  --namespace AWS/DynamoDB \
  --metric-name SuccessfulRequestLatency \
  --dimensions Name=TableName,Value=$TABLE_NAME Name=Operation,Value=PutItem \
  --start-time $START_TIME \
  --end-time $END_TIME \
  --period 300 \
  --statistics Average Maximum \
  --region $REGION > dynamodb_putitem_latency.json

# Consumed Read Capacity
aws cloudwatch get-metric-statistics \
  --namespace AWS/DynamoDB \
  --metric-name ConsumedReadCapacityUnits \
  --dimensions Name=TableName,Value=$TABLE_NAME \
  --start-time $START_TIME \
  --end-time $END_TIME \
  --period 300 \
  --statistics Sum \
  --region $REGION > dynamodb_read_capacity.json

# Consumed Write Capacity
aws cloudwatch get-metric-statistics \
  --namespace AWS/DynamoDB \
  --metric-name ConsumedWriteCapacityUnits \
  --dimensions Name=TableName,Value=$TABLE_NAME \
  --start-time $START_TIME \
  --end-time $END_TIME \
  --period 300 \
  --statistics Sum \
  --region $REGION > dynamodb_write_capacity.json

# User Errors (should be 0)
aws cloudwatch get-metric-statistics \
  --namespace AWS/DynamoDB \
  --metric-name UserErrors \
  --dimensions Name=TableName,Value=$TABLE_NAME \
  --start-time $START_TIME \
  --end-time $END_TIME \
  --period 300 \
  --statistics Sum \
  --region $REGION > dynamodb_errors.json

# Throttled Requests (should be 0)
aws cloudwatch get-metric-statistics \
  --namespace AWS/DynamoDB \
  --metric-name ThrottledRequests \
  --dimensions Name=TableName,Value=$TABLE_NAME \
  --start-time $START_TIME \
  --end-time $END_TIME \
  --period 300 \
  --statistics Sum \
  --region $REGION > dynamodb_throttling.json
```

---

## Screenshot Checklist

From AWS Console:
- [ ] Request latency (GetItem, PutItem)
- [ ] Consumed read/write capacity
- [ ] Throttling events (should be 0)
- [ ] Error rates (should be 0)
- [ ] Operation breakdown (GetItem, PutItem counts)

Optional (if available):
- [ ] CustomerIndex GSI metrics
- [ ] Item count over time
- [ ] Storage size

---

## Expected Metric Values (From 150-Operation Test)

### Request Latency:
- GetItem: ~10-30ms average
- PutItem: ~15-40ms average
- Lower than MySQL (no query planning overhead)

### Consumed Capacity:
- Read capacity: ~150-200 RCUs total
  - 50 GetItem for cart retrieval (Phase 3)
  - 50 GetItem for add-item read-modify-write (Phase 2)
- Write capacity: ~100-150 WCUs total
  - 50 PutItem for create cart (Phase 1)
  - 50 PutItem for add-item updates (Phase 2)

### Throttling:
- **Should be 0** (on-demand mode auto-scales)

### Errors:
- UserErrors: 0 (no client errors)
- SystemErrors: 0 (no DynamoDB errors)

---

## Comparison with MySQL CloudWatch

### MySQL Metrics (for reference):
- CPU Utilization: 12-18%
- Database Connections: 8-12
- Read/Write latency: 2-5ms

### DynamoDB Metrics:
- No CPU (serverless)
- No connections (HTTP API)
- Request latency: 10-30ms (end-to-end)

**Key difference:** DynamoDB shows **request latency** (application → DynamoDB), MySQL shows **query execution time** (inside database).

---

## Time Required
5-10 minutes to capture all screenshots

**Best done immediately after running performance test!**

