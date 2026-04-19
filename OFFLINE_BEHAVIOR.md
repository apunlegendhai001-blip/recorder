# Workflow Behavior When Streamer is Offline

## Scenarios

### Scenario 1: Streamer is Offline from the Start

**What Happens:**
1. Workflow starts and attempts to record
2. goondvr checks if streamer is online (every `interval` minutes, default 1 minute)
3. Finds streamer offline, waits and retries
4. After **30 attempts** (30 minutes with default interval), stops trying
5. Moves to processing job (finds no files, exits gracefully)
6. Restart job triggers after 30-60 seconds
7. Workflow restarts and tries again

**Timeline (with default 1-minute interval):**
```
00:00 - Start recording attempt #1
00:01 - Offline, retry attempt #2
00:02 - Offline, retry attempt #3
...
00:30 - Offline, retry attempt #30
00:30 - Stop trying, proceed to processing
00:31 - Processing finds no files, exits
00:32 - Restart job triggers new workflow run
```

**Configuration:**
- `MAX_INITIAL_ATTEMPTS=30` (30 attempts before giving up)
- With `interval=1`: 30 minutes total
- With `interval=2`: 60 minutes total
- With `interval=5`: 150 minutes total

**Why This Design:**
- Prevents infinite loops when streamer is offline
- Allows workflow to restart and try again
- Doesn't waste GitHub Actions minutes checking forever
- Backup cron schedule ensures regular checks (every 5-6 hours)

---

### Scenario 2: Streamer Goes Offline During Recording

**What Happens:**
1. Recording is active, creating video files
2. Streamer goes offline mid-stream
3. goondvr stops recording, exits
4. Wrapper detects files exist (`RECORDED_SOMETHING=true`)
5. Checks if streamer comes back online
6. After **5 consecutive offline checks**, stops trying
7. Marks all files for upload
8. Processing job uploads the recorded content
9. Restart job triggers to continue monitoring

**Timeline:**
```
00:00 - Recording starts
00:15 - Streamer goes offline
00:15 - Check #1: Offline (wait 1 minute)
00:16 - Check #2: Offline (wait 1 minute)
00:17 - Check #3: Offline (wait 1 minute)
00:18 - Check #4: Offline (wait 1 minute)
00:19 - Check #5: Offline (stop trying)
00:19 - Mark files for upload
00:20 - Processing starts
00:25 - Upload complete
00:26 - Restart triggers
```

**Configuration:**
- `MAX_CONSECUTIVE_OFFLINE=5` (5 checks after recording)
- With `interval=1`: 5 minutes wait
- With `interval=2`: 10 minutes wait
- With `interval=5`: 25 minutes wait

**Why This Design:**
- Gives streamer a chance to come back online
- Doesn't wait too long (5 minutes is reasonable)
- Ensures recorded content is uploaded promptly
- Continues monitoring after upload

---

### Scenario 3: Streamer is Intermittently Online/Offline

**What Happens:**
1. Workflow checks every `interval` minutes
2. If offline: Increments attempt counter
3. If comes online: Records, resets counters
4. If goes offline again: Starts offline counter
5. Process repeats until max attempts reached

**Example Timeline:**
```
00:00 - Attempt #1: Offline
00:01 - Attempt #2: Offline
00:02 - Attempt #3: Online! Start recording
00:10 - Recording... (8 minutes of content)
00:10 - Offline check #1
00:11 - Offline check #2
00:12 - Online again! Continue recording
00:20 - Recording... (8 more minutes)
00:20 - Offline check #1
00:21 - Offline check #2
00:22 - Offline check #3
00:23 - Offline check #4
00:24 - Offline check #5 - Stop and upload
```

**Result:** 16 minutes of content recorded and uploaded

---

### Scenario 4: Long Stream (Multiple Hours)

**What Happens:**
1. Recording starts and continues
2. Files are created in chunks (default 45 minutes each)
3. Background monitor marks completed chunks
4. After 30 minutes of recording, timeout triggers
5. Wrapper restarts recording automatically
6. Process continues until stream ends or 6-hour monitor timeout
7. All files are marked and uploaded

**Timeline:**
```
00:00 - Start recording
00:30 - Timeout, restart recording (seamless)
01:00 - Timeout, restart recording
01:30 - Timeout, restart recording
...
05:00 - Stream ends
05:01 - Offline check #1
05:02 - Offline check #2
05:03 - Offline check #3
05:04 - Offline check #4
05:05 - Offline check #5 - Stop and upload
05:10 - Upload starts (multiple files)
05:30 - Upload complete
```

**Safety Limits:**
- 30-minute timeout per recording attempt
- 6-hour absolute monitor timeout
- 5.5-hour job timeout (330 minutes)
- Auto-restart after job completes

---

## Configuration Options

### Adjust Offline Behavior

**Quick Checks (Faster Detection):**
```yaml
interval: 1  # Check every 1 minute
```
- Detects stream faster
- More API calls
- 30 minutes before giving up if offline
- 5 minutes after stream ends

**Slower Checks (Fewer API Calls):**
```yaml
interval: 5  # Check every 5 minutes
```
- Slower detection (up to 5 minutes delay)
- Fewer API calls
- 150 minutes before giving up if offline
- 25 minutes after stream ends

### Adjust Patience

**More Patient (Wait Longer):**
Edit workflow file:
```bash
MAX_INITIAL_ATTEMPTS=60  # 60 minutes with interval=1
MAX_CONSECUTIVE_OFFLINE=10  # 10 minutes after recording
```

**Less Patient (Give Up Faster):**
```bash
MAX_INITIAL_ATTEMPTS=15  # 15 minutes with interval=1
MAX_CONSECUTIVE_OFFLINE=3  # 3 minutes after recording
```

---

## Monitoring & Logs

### What to Look For

**Streamer Offline from Start:**
```
Starting recording attempt...
Recording exited with code X
Streamer offline (attempt 1/30)
...
Streamer offline (attempt 30/30)
⏸️  Streamer offline after 30 attempts - will retry on next run
```

**Stream Ended:**
```
✓ Found 3 video file(s)
Stream offline check 1/5
Stream offline check 2/5
...
Stream offline check 5/5
🎬 Stream has ended - proceeding to upload
```

**Processing No Files:**
```
No completed files found to process
Debug: Checking videos directory...
Total video files: 0
Total markers: 0
```

---

## Troubleshooting

### Issue: Workflow keeps restarting but never records

**Cause:** Streamer is always offline

**Solution:** This is normal behavior. The workflow will:
1. Check for 30 minutes
2. Give up and restart
3. Repeat until streamer comes online

**Cost:** Minimal - only checking API, not recording

---

### Issue: Workflow stops too quickly when streamer is offline

**Cause:** `MAX_INITIAL_ATTEMPTS` is too low

**Solution:** Increase the value:
```bash
MAX_INITIAL_ATTEMPTS=60  # Wait 60 minutes instead of 30
```

---

### Issue: Workflow waits too long after stream ends

**Cause:** `MAX_CONSECUTIVE_OFFLINE` is too high

**Solution:** Decrease the value:
```bash
MAX_CONSECUTIVE_OFFLINE=3  # Wait 3 minutes instead of 5
```

---

### Issue: Missing the start of streams

**Cause:** `interval` is too high

**Solution:** Use faster checking:
```yaml
interval: 1  # Check every minute
```

---

## Best Practices

### For Active Streamers (Stream Daily)
```yaml
interval: 1  # Fast detection
MAX_INITIAL_ATTEMPTS: 30  # 30 minutes patience
MAX_CONSECUTIVE_OFFLINE: 5  # 5 minutes after stream
```

### For Irregular Streamers (Stream Occasionally)
```yaml
interval: 2  # Moderate detection
MAX_INITIAL_ATTEMPTS: 60  # 2 hours patience
MAX_CONSECUTIVE_OFFLINE: 10  # 20 minutes after stream
```

### For Rare Streamers (Stream Rarely)
```yaml
interval: 5  # Slow detection
MAX_INITIAL_ATTEMPTS: 120  # 10 hours patience
MAX_CONSECUTIVE_OFFLINE: 15  # 75 minutes after stream
```

---

## Summary

| Scenario | Behavior | Time to Give Up |
|----------|----------|-----------------|
| Offline from start | Check 30 times, then restart | 30 minutes (interval=1) |
| Goes offline during | Check 5 times, then upload | 5 minutes (interval=1) |
| Long stream | Record until ends or 6h timeout | 6 hours max |
| Intermittent | Record when online, wait when offline | Varies |

**Key Points:**
- ✅ Never loops forever
- ✅ Always progresses to next step
- ✅ Uploads recorded content even if stream ends
- ✅ Automatically restarts to continue monitoring
- ✅ Configurable patience levels
- ✅ Backup cron schedule ensures regular checks
