# Edge Cases & Bugs Fixed

## Critical Bugs Fixed

### 1. **Empty/Corrupted File Handling**
- **Bug**: Empty or corrupted .ts files would be processed, causing FFmpeg to fail
- **Fix**: Added file size validation (minimum 1KB) before processing
- **Impact**: Prevents wasted processing time and failed uploads

### 2. **Missing MP4 Verification**
- **Bug**: If FFmpeg succeeded but didn't create MP4, upload would fail silently
- **Fix**: Verify MP4 file exists and is non-empty before upload attempt
- **Impact**: Catches conversion failures early

### 3. **Duplicate File Marking**
- **Bug**: Background monitor could mark files multiple times
- **Fix**: Check if marker already exists before creating
- **Impact**: Prevents duplicate processing

### 4. **GoFile Server Fallback**
- **Bug**: If server API fails, upload would crash
- **Fix**: Proper fallback to "store1" with null checks
- **Impact**: More resilient uploads

### 5. **Math Operations on Empty Values**
- **Bug**: `bc` calculations fail on empty/null values from jq
- **Fix**: Added error handling and default values (0)
- **Impact**: Prevents script crashes in summary calculations

### 6. **Artifact Chunk Upload Bug**
- **Bug**: Chunk upload step referenced undefined `${{ matrix.chunk }}` variable
- **Fix**: Simplified to delete oldest files instead of complex chunking
- **Impact**: Actually works now instead of failing silently

### 7. **Empty Secret Handling**
- **Bug**: Empty COOKIES or USER_AGENT secrets would cause silent failures
- **Fix**: Validate secrets and provide warnings/defaults
- **Impact**: Better debugging and fallback behavior

### 8. **Thundering Herd on Restart**
- **Bug**: All channels restart simultaneously, overwhelming GitHub API
- **Fix**: Random delay (30-60s) before restart
- **Impact**: Reduces API rate limit issues

## Edge Cases Handled

### Recording Phase

1. **Insufficient Disk Space at Start**
   - Check: Requires 3GB minimum before starting
   - Action: Fail fast with clear error message

2. **Disk Space Exhaustion During Recording**
   - Monitor: Check every minute
   - Warning: At 2GB remaining
   - Emergency Stop: At 1GB to prevent job failure

3. **Empty Files from Recording Errors**
   - Check: File size before marking as complete
   - Action: Skip empty files, delete them

4. **Resume from Previous Run**
   - Detect: Count existing .ts files
   - Action: Continue recording, mark completed files

5. **Timeout Reached (5 hours)**
   - Detect: Exit code 124 from timeout command
   - Action: Mark files and trigger restart

### Artifact Upload Phase

6. **Artifact Size Approaching 10GB Limit**
   - Check: Calculate total size before upload
   - Warning: At 8GB
   - Action: Delete oldest files to stay under 8GB

7. **No Files to Upload**
   - Check: Verify videos directory exists and has files
   - Action: Skip upload gracefully

### Processing Phase

8. **Missing .ts File for .uploaded Marker**
   - Check: Verify .ts file exists
   - Action: Delete orphaned marker, continue

9. **Corrupted .ts Files**
   - Check: Minimum 1KB size
   - Action: Delete both .ts and marker

10. **FFmpeg Conversion Timeout**
    - Timeout: 30 minutes per file
    - Action: Remove marker for retry, count as failed

11. **FFmpeg Creates Empty MP4**
    - Check: MP4 exists and >1KB
    - Action: Delete MP4, remove marker for retry

12. **Large File Upload Warning**
    - Check: Files >5GB
    - Action: Warn but proceed (may be slow)

13. **GoFile Upload Timeout**
    - Timeout: 60 minutes per attempt
    - Retry: 3 attempts with 5s delay
    - Action: Keep files for next run if all fail

14. **GoFile Server API Failure**
    - Fallback: Use "store1" if API unreachable
    - Validation: Check for null/empty responses

15. **Network Errors During Upload**
    - Detect: HTTP code 000 (timeout/network)
    - Action: Retry with exponential backoff

16. **Low Disk Space During Processing**
    - Check: Before each file conversion
    - Action: Skip file if <1GB available

17. **Database Corruption**
    - Check: Validate JSON before operations
    - Fallback: Create new database if missing

### Restart Phase

18. **GitHub API Rate Limiting**
    - Random Delay: 30-60 seconds
    - Retry: 3 attempts with 5s delay
    - Fallback: Rely on cron schedule

19. **Restart Failure**
    - Fallback: Cron runs every 5 hours as backup
    - Log: Clear message about next scheduled run

### Data Integrity

20. **Processing Order**
    - Sort: Process files in chronological order
    - Limit: 50 files per run to prevent timeout

21. **Concurrent Processing**
    - Limit: Max 5 channels processing simultaneously
    - Reason: Prevent resource exhaustion

22. **Failed Upload Retry**
    - Keep: .ts and .mp4 files if upload fails
    - Retry: Next processing run will attempt again

23. **Successful Upload Cleanup**
    - Delete: .ts, .mp4, and .uploaded marker
    - Verify: All three deleted to prevent orphans

## Monitoring & Observability

### Added Metrics
- Disk space usage (GB and %)
- File sizes (bytes, human-readable)
- Conversion duration (seconds)
- Upload duration (seconds)
- Upload speed (MB/s)
- Processed/failed counts
- Total uploaded size (GB)

### Status Messages
- ✓ Success indicators
- ⚠️  Warnings
- ✗ Failure indicators
- 🛑 Critical errors
- ⏱️  Timeout notifications
- 🔀 Split/chunk operations

## Configuration Changes

### Chunk Size
- **Before**: 240 minutes (4 hours)
- **After**: 45 minutes
- **Reason**: Smaller files, safer artifact sizes

### Processing Limit
- **Before**: 100 files per run
- **After**: 50 files per run
- **Reason**: Prevent processing timeout

### Timeouts Added
- FFmpeg conversion: 30 minutes
- GoFile upload: 60 minutes
- Recording: 5 hours (existing)
- Processing job: 6 hours (existing)

### Retry Logic
- Upload attempts: 3 retries
- Restart attempts: 3 retries
- Delay between retries: 5 seconds

## Testing Recommendations

1. Test with empty secrets
2. Test with disk space <3GB
3. Test with corrupted .ts files
4. Test with >10GB of recordings
5. Test GoFile API failure
6. Test network timeout during upload
7. Test FFmpeg timeout on large files
8. Test restart with 20+ channels
9. Test resume after timeout
10. Test processing with 100+ files

## Known Limitations

1. **Artifact chunking**: Simplified to delete old files instead of true chunking
2. **GoFile rate limits**: No handling for API rate limits
3. **Concurrent uploads**: No queue management for many large files
4. **Database backups**: No automatic backup rotation
5. **Failed file tracking**: No persistent retry queue across runs
