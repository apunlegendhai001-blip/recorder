# Workflow Hanging/Blocking Issues Analysis

## Issues Found & Fixed

### ✅ Issue #1: Recording Never Stops When Stream Ends (FIXED)
**Problem**: The goondvr app keeps checking indefinitely when streamer goes offline, preventing the workflow from moving to processing step.

**Root Cause**: 
- App runs in infinite loop checking if streamer is online
- No mechanism to detect "stream has ended, time to process files"
- Workflow waits forever for app to exit

**Solution Implemented**:
1. **Recording Wrapper with Timeout**: 30-minute timeout per check
2. **Offline Detection**: Tracks consecutive offline checks (max 5)
3. **Auto-Stop**: Stops after 5 consecutive offline checks when files exist
4. **Background Monitor**: Detects when no new files are being created

**Code**:
```bash
# Wrapper that monitors for stream end
RECORDED_SOMETHING=false
CONSECUTIVE_OFFLINE=0
MAX_CONSECUTIVE_OFFLINE=5

while true; do
  timeout 30m ./goondvr ...
  
  if [ "$RECORDED_SOMETHING" = "true" ]; then
    CONSECUTIVE_OFFLINE=$((CONSECUTIVE_OFFLINE + 1))
    if [ $CONSECUTIVE_OFFLINE -ge $MAX_CONSECUTIVE_OFFLINE ]; then
      echo "🎬 Stream has ended - proceeding to upload"
      break
    fi
  fi
done
```

---

### ⚠️ Issue #2: Background Monitor Can Run Forever
**Problem**: The background monitor process runs in infinite loop and might not stop properly.

**Current State**: Partially fixed
- Has pkill mechanism
- Has break conditions
- But might not always trigger

**Potential Issues**:
1. Monitor checks every 60 seconds
2. If main process hangs, monitor keeps running
3. Could prevent job from completing

**Recommendation**: Add timeout to background monitor

**Proposed Fix**:
```bash
(
  START_TIME=$(date +%s)
  MAX_MONITOR_TIME=$((6 * 3600))  # 6 hours max
  
  while true; do
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))
    
    if [ $ELAPSED -gt $MAX_MONITOR_TIME ]; then
      echo "⏱️  Monitor timeout reached - stopping"
      pkill -f goondvr || true
      break
    fi
    
    # ... rest of monitor logic
  done
) &
```

---

### ⚠️ Issue #3: FlareSolverr Wait Can Hang
**Problem**: Waits up to 60 seconds for FlareSolverr, but no timeout on the test request.

**Current Code**:
```bash
for i in {1..30}; do
  if curl -s http://localhost:8191/v1 > /dev/null 2>&1; then
    break
  fi
  sleep 2
done

# Test FlareSolverr (NO TIMEOUT!)
curl -s -X POST http://localhost:8191/v1 ...
```

**Risk**: Test request could hang indefinitely

**Proposed Fix**:
```bash
# Add timeout to test request
curl -s --max-time 30 -X POST http://localhost:8191/v1 ...
```

---

### ⚠️ Issue #4: File Marking Loop Can Block
**Problem**: The file marking loop in background monitor uses a subshell that could block.

**Current Code**:
```bash
find videos -type f \( -name "*.ts" -o -name "*.mp4" \) 2>/dev/null | while read VIDEO_FILE; do
  # Check if file is still being written
  SIZE1=$(stat -c%s "$VIDEO_FILE" 2>/dev/null || echo "0")
  sleep 2  # <-- BLOCKS for 2 seconds per file!
  SIZE2=$(stat -c%s "$VIDEO_FILE" 2>/dev/null || echo "0")
done
```

**Risk**: If there are many files, this loop takes a long time (2 seconds × number of files)

**Proposed Fix**:
```bash
# Process only unmarked files and limit to 10 per iteration
find videos -type f \( -name "*.ts" -o -name "*.mp4" \) 2>/dev/null | head -10 | while read VIDEO_FILE; do
  MARKER="${VIDEO_FILE}.uploaded"
  if [ -f "$MARKER" ]; then
    continue  # Skip already marked files
  fi
  # ... rest of logic
done
```

---

### ⚠️ Issue #5: Processing Job Can Hang on Large Files
**Problem**: FFmpeg conversion has 30-minute timeout, but upload has 60-minute timeout. Very large files could cause issues.

**Current State**: Has timeouts
- FFmpeg: 30 minutes
- Upload: 60 minutes

**Potential Issue**: 
- 5GB file at slow speed could take >60 minutes
- Timeout kills curl but doesn't clean up properly

**Recommendation**: Add progress monitoring

**Proposed Fix**:
```bash
# Add progress callback for large files
if (( $(echo "$MP4_SIZE_GB > 3" | bc -l) )); then
  echo "⚠️  Large file detected - upload may take a while"
  echo "Estimated time: ~$((MP4_SIZE_GB * 10)) minutes at 10MB/s"
fi

# Monitor upload progress in background
(
  while kill -0 $UPLOAD_PID 2>/dev/null; do
    sleep 30
    echo "Upload still in progress..."
  done
) &
```

---

### ⚠️ Issue #6: Restart Job Might Not Trigger
**Problem**: Restart job depends on both record and process jobs completing. If either hangs, restart never happens.

**Current Condition**:
```yaml
if: "!startsWith(github.ref, 'refs/tags/') && (success() || failure()) && needs.setup.outputs.has-channels == 'true'"
needs: [setup, record, process]
```

**Risk**: 
- If record job times out (330 min), restart waits
- If process job times out (360 min), restart waits
- Total possible wait: 690 minutes (11.5 hours)

**Recommendation**: Add separate restart trigger

**Proposed Fix**:
```yaml
# Add a separate scheduled restart that doesn't depend on jobs
- cron: '0 */6 * * *'  # Every 6 hours as backup
```

---

### ⚠️ Issue #7: Database Lock Can Deadlock
**Problem**: Using flock for database operations, but no timeout on lock acquisition.

**Current Code**:
```bash
(
  flock -x 200  # <-- Waits forever if lock is held
  # ... database operations
) 200>database/.lock
```

**Risk**: If a previous process crashed while holding lock, all future operations hang

**Proposed Fix**:
```bash
(
  # Wait max 30 seconds for lock
  if flock -x -w 30 200; then
    # ... database operations
  else
    echo "✗ Failed to acquire database lock after 30 seconds"
    exit 1
  fi
) 200>database/.lock
```

---

### ⚠️ Issue #8: Artifact Upload Can Hang
**Problem**: No timeout on artifact upload action.

**Current Code**:
```yaml
- uses: actions/upload-artifact@v4
  with:
    name: raw-videos-...
    path: videos/**/*.ts
```

**Risk**: Large artifacts (approaching 10GB) could take very long or hang

**Recommendation**: Monitor artifact upload time

**Proposed Fix**: Add a step before upload to estimate time:
```yaml
- name: Estimate upload time
  run: |
    SIZE_GB=$(du -sb videos | awk '{print $1/1024/1024/1024}')
    EST_MINUTES=$(echo "$SIZE_GB * 2" | bc)  # ~2 min per GB
    echo "Estimated upload time: ${EST_MINUTES} minutes"
    
    if (( $(echo "$EST_MINUTES > 30" | bc -l) )); then
      echo "⚠️  WARNING: Upload may take over 30 minutes"
    fi
```

---

### ⚠️ Issue #9: No Overall Job Timeout
**Problem**: Individual steps have timeouts, but the overall record job could theoretically run for 330 minutes even if stuck.

**Current State**: 
- Record job: 330 minutes timeout
- Process job: 360 minutes timeout

**Risk**: Job could be stuck in a loop within the timeout

**Recommendation**: Add heartbeat monitoring

**Proposed Fix**:
```bash
# Add heartbeat file
HEARTBEAT_FILE="/tmp/workflow_heartbeat"
touch "$HEARTBEAT_FILE"

# Update heartbeat in background
(
  while true; do
    sleep 60
    touch "$HEARTBEAT_FILE"
  done
) &
HEARTBEAT_PID=$!

# Monitor heartbeat
(
  while true; do
    sleep 300  # Check every 5 minutes
    
    # Check if heartbeat is stale
    if [ -f "$HEARTBEAT_FILE" ]; then
      AGE=$(($(date +%s) - $(stat -c%Y "$HEARTBEAT_FILE")))
      if [ $AGE -gt 600 ]; then  # 10 minutes stale
        echo "🛑 Heartbeat stale - workflow appears hung"
        pkill -f goondvr || true
        break
      fi
    fi
  done
) &
```

---

### ⚠️ Issue #10: Wait Command Can Block Forever
**Problem**: Using `wait` without timeout.

**Current Code**:
```bash
kill $UPLOADER_PID 2>/dev/null || true
wait $UPLOADER_PID 2>/dev/null || true  # <-- Could wait forever
```

**Risk**: If process is stuck in uninterruptible sleep, wait hangs

**Proposed Fix**:
```bash
# Kill and wait with timeout
kill $UPLOADER_PID 2>/dev/null || true
sleep 2

# Force kill if still running
if kill -0 $UPLOADER_PID 2>/dev/null; then
  echo "Process didn't stop, force killing..."
  kill -9 $UPLOADER_PID 2>/dev/null || true
fi

# Don't wait, just continue
```

---

## Summary of Recommendations

### Critical (Implement Now)
1. ✅ Add timeout to recording wrapper (DONE)
2. ✅ Add offline detection (DONE)
3. ⚠️ Add timeout to FlareSolverr test
4. ⚠️ Add timeout to database lock (flock -w 30)
5. ⚠️ Remove blocking wait, use force kill

### High Priority
6. ⚠️ Add timeout to background monitor (6 hours max)
7. ⚠️ Limit file marking loop (max 10 files per iteration)
8. ⚠️ Add backup restart trigger (cron every 6 hours)

### Medium Priority
9. ⚠️ Add upload progress monitoring for large files
10. ⚠️ Add heartbeat monitoring for overall job health
11. ⚠️ Add artifact upload time estimation

### Low Priority
12. Add metrics collection for debugging
13. Add workflow run duration tracking
14. Add notification on long-running jobs

## Testing Checklist

- [ ] Test stream ending detection (5 consecutive offline checks)
- [ ] Test background monitor stops properly
- [ ] Test FlareSolverr timeout
- [ ] Test database lock timeout
- [ ] Test force kill of stuck processes
- [ ] Test large file upload (>5GB)
- [ ] Test artifact upload timeout
- [ ] Test restart trigger after timeout
- [ ] Test concurrent processing with locks
- [ ] Test overall job timeout (330 min)

## Monitoring Recommendations

Add these log messages to track progress:
```bash
echo "[$(date +%s)] CHECKPOINT: Recording started"
echo "[$(date +%s)] CHECKPOINT: Files marked"
echo "[$(date +%s)] CHECKPOINT: Upload started"
echo "[$(date +%s)] CHECKPOINT: Processing complete"
```

This allows tracking where the workflow might be stuck.
