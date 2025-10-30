# 📸 CloudWatch Screenshots - Complete Guide

## Prerequisites
- AWS Console open in browser
- Previous test data (from ~7:18 AM test)
- No need to run new tests!

---

## 🎯 STEP 1: Switch to Correct Region

**CRITICAL: Your resources are in us-west-2!**

1. Look at the **TOP RIGHT** corner of AWS Console
2. You'll see a region name (might say "N. Virginia" or "us-east-1")
3. Click on it to open the dropdown
4. Scroll down and select: **"US West (Oregon)"** or **"us-west-2"**
5. Wait for page to refresh

✅ Verify: Top right should now show "Oregon" or "us-west-2"

---

## 🎯 STEP 2: Navigate to CloudWatch

1. Click the **search bar** at the very top (says "Search AWS")
2. Type: `CloudWatch`
3. Click on **"CloudWatch"** under Services
4. You should see the CloudWatch dashboard

Alternative: Use direct link
```
https://us-west-2.console.aws.amazon.com/cloudwatch/home?region=us-west-2
```

---

## 🎯 STEP 3: Go to Metrics Section

1. In the **left sidebar**, find and click **"Metrics"**
2. Under Metrics, click **"All metrics"**
3. You'll see the Metrics Explorer interface

---

## 🎯 STEP 4: Select SQS Metrics

In the main area, you'll see "Browse" tab:

1. Look for **"AWS namespaces"** section
2. Find and click **"SQS"** (shows Amazon SQS icon)
3. Click on **"Queue Metrics"**
4. You'll see a table with metrics for "order-processing-queue"

---

## 🎯 STEP 5: Select the Metrics You Need

In the metrics table, find and **CHECK** these boxes:

- ✅ **ApproximateNumberOfMessagesVisible** (Queue depth)
- ✅ **ApproximateNumberOfMessagesNotVisible** (In-flight messages)

The graph will automatically appear above the table!

---

## 🎯 STEP 6: Adjust Time Range

At the **top right** of the graph:

1. Click the time selector (shows something like "1h" or "3h")
2. Select **"Custom"**
3. Choose **"Relative"**
4. Select **"Last 3 hours"**
5. Click **"Apply"**

This will show your test data from earlier today.

---

## 🎯 STEP 7: Take Screenshots

### Screenshot 1: Full Queue Lifecycle
**File name:** `cloudwatch_full_queue_lifecycle.png`

**What to capture:**
- Time range: Last 3 hours
- Both metrics visible (ApproximateNumberOfMessagesVisible and NotVisible)
- Should show:
  - Baseline at 0
  - Sharp spike during flash sale (~2,880 messages)
  - Slow decrease as worker processes backlog

**How to capture:**
- **Mac:** Press `Cmd + Shift + 4`, drag to select area
- **Windows:** Press `Windows + Shift + S`, drag to select area
- **Chrome Extension:** Right-click → "Capture screenshot"

---

### Screenshot 2: Flash Sale Spike (Zoomed In)
**File name:** `cloudwatch_flash_sale_spike.png`

**What to do:**
1. Adjust time range to show just 10-minute window around your test
2. In the time selector, use **"Absolute"** instead of Relative
3. Set start time: Around 7:15 AM (before test)
4. Set end time: Around 7:30 AM (after test)
5. This zooms into just the flash sale period

**What you should see:**
- Very sharp upward slope (46.6 messages/second increase)
- Peak at ~2,880-2,900 messages
- Start of downward slope

**Take screenshot**

---

### Screenshot 3: Queue Drain Period
**File name:** `cloudwatch_queue_drain.png`

**What to do:**
1. Adjust time range to show 30-60 minutes after test
2. Set start time: 7:20 AM (after test started)
3. Set end time: 8:00 AM (after drain period)

**What you should see:**
- Slow, steady decrease in queue depth
- Much gentler slope than the spike
- Takes 20-30 minutes to drain
- Eventually reaches near zero (or your current 2,734 if worker stopped)

**Take screenshot**

---

### Screenshot 4 (Optional): In-Flight Messages Detail
**File name:** `cloudwatch_in_flight_messages.png`

**What to focus on:**
- Make sure "ApproximateNumberOfMessagesNotVisible" is visible
- Should show a flat line around 5 messages
- This proves the payment processor bottleneck (5-slot semaphore)

**Take screenshot**

---

## ✅ WHAT YOU SHOULD SEE

### Correct Patterns:

**ApproximateNumberOfMessagesVisible (Orange/Blue line):**
```
📈 Sharp upward spike during test (60 seconds)
📊 Peak at ~2,880 messages
📉 Slow downward slope (20-30 minutes to drain)
```

**ApproximateNumberOfMessagesNotVisible (Different color):**
```
⏸️ Flat line at ~5 messages (or 0 if worker stopped)
⏸️ This represents messages being processed
⏸️ Limited by payment processor semaphore
```

---

## ❌ TROUBLESHOOTING

### Problem: "No data available"
**Solution:**
- Check region (must be us-west-2)
- Check time range (must include test time ~7:18 AM)
- Verify queue name: "order-processing-queue"

### Problem: "Graph shows all zeros"
**Solution:**
- Wrong time range - adjust to when you ran tests
- Check if metrics are selected (checkboxes)

### Problem: "Can't find SQS in metrics"
**Solution:**
- Make sure you're in "All metrics" tab
- Look in "AWS namespaces" section
- Queue must have received messages to appear

### Problem: "Queue shows different pattern"
**Solution:**
- If instantly zero: Worker processed everything (unlikely)
- If steadily high: Worker might still be running
- If no spike: Test might not have run properly

---

## 📊 EXPECTED METRICS VALUES

Based on your actual test:

| Metric | Value |
|--------|-------|
| Peak Queue Depth | ~2,880 messages |
| Queue Growth Rate | 46.6 messages/second |
| Queue Drain Rate | 1.67 messages/second |
| Time to Peak | 60 seconds |
| Time to Drain | 20-30 minutes |
| In-Flight Messages | 0-5 messages |

---

## 🔄 ALTERNATIVE: Run Fresh Test (If Needed)

If you want fresh data with perfect timestamps:

### Step 1: Start Server and Worker
```bash
# Terminal 1: Main server
cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/src
./server > /tmp/server.log 2>&1 &

# Terminal 2: Worker
cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/src
WORKER_MODE=true NUM_WORKERS=1 ./server > /tmp/worker.log 2>&1 &
```

### Step 2: Run Flash Sale Test
```bash
cd /Users/ayushjain/Documents/Grad/CS6620/CS6650_2b_demo/locust
source .venv/bin/activate
ASYNC_MODE=true locust -f locustfile_orders.py \
  --host http://localhost:8080 \
  -u 20 -r 10 -t 60s --headless --only-summary
```

### Step 3: Note the Time
- Write down: Test started at [TIME]
- Use this time when adjusting CloudWatch time range

### Step 4: Get Screenshots
- Follow steps above
- Use the exact time you just noted
- Should see fresh spike in metrics

### Step 5: Stop Services
```bash
pkill -f './server'
```

---

## 💡 PRO TIPS

1. **Use "Custom" time range** for precise control
2. **Graph statistic:** Set to "Maximum" for clearer peaks
3. **Period:** Use 1-minute intervals for detailed view
4. **Export data:** Can download CSV from CloudWatch
5. **Multiple graphs:** Can add more metrics for comparison

---

## 📋 SCREENSHOT CHECKLIST

Before submitting, verify you have:

- ✅ Screenshot showing full queue lifecycle (3-hour view)
- ✅ Screenshot showing flash sale spike (zoomed in)
- ✅ Screenshot showing queue drain period
- ✅ Optional: In-flight messages showing bottleneck
- ✅ All screenshots clearly labeled with filename
- ✅ Timestamps visible in screenshots
- ✅ Both metrics (Visible and NotVisible) shown

---

## 🎓 WHAT THESE SCREENSHOTS PROVE

Your screenshots will demonstrate:

1. **Queue buffering works** - Spike shows all orders accepted
2. **Processing bottleneck exists** - Slow drain proves limited capacity
3. **Worker scaling needed** - 27-minute drain is unacceptable
4. **Async architecture success** - No orders lost despite spike
5. **Business case for scaling** - Visual proof of need for more workers

These screenshots are KEY EVIDENCE for your report!

---

## 📞 NEED HELP?

If stuck:
1. Verify region: us-west-2
2. Check time range matches test time
3. Ensure queue name: "order-processing-queue"
4. Look for data around 7:18 AM (your test time)

Current queue still has 2,734 messages - proves the backlog exists!

---

Good luck! These screenshots will make your report very compelling. 🎉

