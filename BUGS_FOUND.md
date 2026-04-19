# Critical Bugs Found in Workflow

## 🐛 Bug #1: Monitoring Only Shows .ts Files (CRITICAL)
**Location**: Record job, background monitor
**Line**: ~290

```yaml
find videos -type f -name "*.ts" -exec ls -lh {} \; 2>/dev/null || echo "No files yet"
```

**Problem**: The status monitor only shows `.ts` files, but the app can create `.mp4` files. This gives misleading status information.

**Impact**: User can't see `.mp4` recordings in progress

**Fix**: Change to:
```yaml
find videos -type f \( -name "*.ts" -o -name "*.mp4" \) -exec ls -lh {} \; 2>/dev/null || echo "No files yet"
```

---

## 🐛 Bug #2: Indentation Error in Processing Loop (CRITICAL)
**Location**: Process job, line ~680

```yaml
              fi
            fi  # <-- EXTRA closing brace!
          done
```

**Problem**: There's an extra `fi` statement that will cause bash syntax error. The conversion block has:
- `if [ "$FILE_EXT" = "mp4" ]; then` ... `else` ... `fi`
- Then another `fi` that doesn't match any `if`

**Impact**: Processing job will fail with syntax error

**Fix**: Remove the extra `fi` after the conversion block

---

## 🐛 Bug #3: Missing `bc` Package (HIGH)
**Location**: Multiple places using `bc` for calculations

**Problem**: The workflow uses `bc` for floating-point math but never installs it:
```bash
TOTAL_SIZE_GB=$(echo "scale=2; $TOTAL_SIZE / 1024 / 1024 / 1024" | bc)
```

**Impact**: All size calculations will fail silently

**Fix**: Add to setup steps:
```yaml
- name: Install dependencies
  run: sudo apt-get update && sudo apt-get install -y ffmpeg bc
```

---

## 🐛 Bug #4: Filename Mismatch in Database Record (MEDIUM)
**Location**: Process job, database record creation

```yaml
--arg filename "$BASENAME.mp4" \
```

**Problem**: Always saves as `.mp4` in database even if original was `.ts` and conversion failed. Should use actual MP4 filename.

**Impact**: Database has incorrect filenames

**Fix**: Use:
```yaml
--arg filename "$(basename "$MP4_FILE")" \
```

---

## 🐛 Bug #5: stat Command Platform Inconsistency (MEDIUM)
**Location**: Multiple places

```bash
stat -c%s "$FILE"  # Linux
stat -f%z "$FILE"  # macOS
```

**Problem**: GitHub Actions uses Ubuntu (Linux), so the macOS fallback will never work. The order should be reversed or just use Linux version.

**Impact**: Unnecessary fallback code that never executes

**Fix**: Since it's always Ubuntu, just use:
```bash
stat -c%s "$FILE" 2>/dev/null || echo "0"
```

---

## 🐛 Bug #6: Missing jq Installation Check (LOW)
**Location**: Record job, settings.json creation

```bash
jq -n --arg cookies "$CLEAN_COOKIES" ...
```

**Problem**: Uses `jq` without verifying it's installed. Processing job checks for jq, but record job doesn't.

**Impact**: Settings creation could fail silently

**Fix**: Add check or install jq in setup

---

## 🐛 Bug #7: Artifact Download Path Issue (MEDIUM)
**Location**: Process job, download artifacts

```yaml
pattern: raw-videos-${{ matrix.channel.username }}-${{ github.run_number }}*
path: videos/
merge-multiple: true
```

**Problem**: When merging multiple artifacts, files might end up in nested directories like `videos/videos/` depending on how they were uploaded.

**Impact**: Processing job might not find files

**Fix**: Verify artifact structure or flatten on download

---

## 🐛 Bug #8: Cache Key Collision (LOW)
**Location**: Record and Process jobs

Record saves:
```yaml
key: state-${{ matrix.channel.username }}-resume-${{ github.run_id }}
```

Process restores:
```yaml
key: state-${{ matrix.channel.username }}-${{ github.run_id }}
restore-keys: state-${{ matrix.channel.username }}-
```

**Problem**: Process job won't find the cache saved by record job because keys don't match exactly.

**Impact**: Database state not shared between jobs

**Fix**: Use consistent key pattern or add `-resume-` to restore-keys

---

## 🐛 Bug #9: CONVERT_DURATION Undefined for .mp4 Files (MEDIUM)
**Location**: Process job, database record

```yaml
if [ "$FILE_EXT" = "mp4" ]; then
  echo "File is already MP4, skipping conversion"
  MP4_FILE="$VIDEO_FILE"
  CONVERT_DURATION=0
else
  # conversion...
fi

# Later used in database record
--arg convert_duration "$CONVERT_DURATION"
```

**Problem**: If the `else` block fails, `CONVERT_DURATION` is never set, causing the database record to fail.

**Impact**: Database insertion fails for failed conversions

**Fix**: Initialize at the start:
```bash
CONVERT_DURATION=0
```

---

## 🐛 Bug #10: No Cleanup of processed/ Directory (MEDIUM)
**Location**: Process job

**Problem**: Creates `processed/` directory for temporary MP4 files but never cleans it up. Over multiple runs, this could fill disk.

**Impact**: Disk space waste

**Fix**: Add cleanup at end or use temp directory

---

## 🐛 Bug #11: GoFile API Rate Limiting Not Handled (HIGH)
**Location**: Process job, upload section

**Problem**: No handling for GoFile API rate limits (HTTP 429). Will just fail and retry with same timing.

**Impact**: Repeated failures if rate limited

**Fix**: Add exponential backoff and detect 429 responses

---

## 🐛 Bug #12: Concurrent Processing Race Condition (MEDIUM)
**Location**: Process job with max-parallel: 5

**Problem**: Multiple channels processing simultaneously could try to write to same database file, causing corruption.

**Impact**: Database corruption

**Fix**: Add file locking or use separate database files per channel

---

## 🐛 Bug #13: Missing Error Handling for jq Operations (MEDIUM)
**Location**: Multiple places

```bash
jq ".records += [$RECORD]" database/uploads.json > database/uploads.json.tmp
mv database/uploads.json.tmp database/uploads.json
```

**Problem**: If jq fails, the temp file might be empty or invalid, and mv will overwrite the good database.

**Impact**: Database loss

**Fix**: Check jq exit code before mv:
```bash
if jq ".records += [$RECORD]" database/uploads.json > database/uploads.json.tmp; then
  mv database/uploads.json.tmp database/uploads.json
else
  rm -f database/uploads.json.tmp
fi
```

---

## 🐛 Bug #14: Restart Job Doesn't Handle Branch Detection (LOW)
**Location**: Restart job

```javascript
ref: context.ref.replace('refs/heads/', '') || 'main'
```

**Problem**: If triggered by schedule (no ref), this might fail or use wrong branch.

**Impact**: Restart might not work from scheduled runs

**Fix**: Add better branch detection

---

## 🐛 Bug #15: No Validation of GoFile Response Structure (HIGH)
**Location**: Process job, upload response parsing

```bash
DOWNLOAD_LINK=$(echo "$RESPONSE_BODY" | jq -r '.data.downloadPage' 2>/dev/null || echo "")
```

**Problem**: Doesn't validate that response has expected structure. GoFile could change API format.

**Impact**: Silent failures with no error message

**Fix**: Validate response structure:
```bash
if echo "$RESPONSE_BODY" | jq -e '.status == "ok"' >/dev/null 2>&1; then
  DOWNLOAD_LINK=$(echo "$RESPONSE_BODY" | jq -r '.data.downloadPage')
else
  ERROR_MSG=$(echo "$RESPONSE_BODY" | jq -r '.status' 2>/dev/null || echo "unknown")
  echo "✗ GoFile API error: $ERROR_MSG"
fi
```

---

## Summary

**Critical (Must Fix)**: 2
- Bug #1: Monitoring only shows .ts files
- Bug #2: Indentation error (syntax error)

**High Priority**: 3
- Bug #3: Missing bc package
- Bug #11: No rate limit handling
- Bug #15: No response validation

**Medium Priority**: 7
- Bugs #4, #5, #7, #8, #9, #10, #12, #13

**Low Priority**: 3
- Bugs #6, #8, #14
