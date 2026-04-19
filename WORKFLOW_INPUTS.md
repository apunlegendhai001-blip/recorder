# Workflow Manual Trigger Options

When manually triggering the workflow via the "Run workflow" button on GitHub Actions, you can customize the recording parameters.

## How to Use

1. Go to **Actions** tab in your GitHub repository
2. Select **GoondVR** workflow from the left sidebar
3. Click **Run workflow** button (top right)
4. Configure the optional parameters
5. Click **Run workflow** to start

## Available Parameters

### 📹 Recording Chunk Duration
**Parameter**: `max_duration`  
**Description**: How long each recording file should be before starting a new file  
**Default**: 45 minutes  
**Options**:
- 15 minutes - Smallest chunks, safest for artifact limits
- 30 minutes - Small chunks
- **45 minutes** - Default, balanced
- 60 minutes - 1 hour chunks
- 90 minutes - 1.5 hour chunks
- 120 minutes - 2 hour chunks
- 180 minutes - 3 hour chunks
- 240 minutes - 4 hour chunks (maximum recommended)

**When to change**:
- Use **15-30 minutes** if you're hitting artifact size limits
- Use **60-120 minutes** if you want fewer files to manage
- Keep **45 minutes** for best balance

### 🎬 Resolution
**Parameter**: `resolution`  
**Description**: Target video resolution  
**Default**: 9999 (source quality)  
**Options**:
- **9999** - Source quality (no downscaling)
- 1080 - 1080p (Full HD)
- 720 - 720p (HD)
- 480 - 480p (SD)

**When to change**:
- Use **1080** or **720** to reduce file sizes
- Use **9999** to keep original quality

### 🎞️ Framerate
**Parameter**: `framerate`  
**Description**: Target video framerate  
**Default**: 60 fps  
**Options**:
- **60** - 60fps (smooth, source quality)
- 30 - 30fps (standard)
- 24 - 24fps (cinematic)

**When to change**:
- Use **30** to reduce file sizes
- Use **60** for smooth motion

### ⏱️ Check Interval
**Parameter**: `interval`  
**Description**: How often to check if streamer is online when offline  
**Default**: 1 minute  
**Options**:
- **1** - Check every minute (fastest detection)
- 2 - Check every 2 minutes
- 5 - Check every 5 minutes
- 10 - Check every 10 minutes

**When to change**:
- Use **1** for fastest stream detection
- Use **5-10** to reduce API calls

### 🐛 Debug Logging
**Parameter**: `debug`  
**Description**: Enable detailed debug output  
**Default**: false  
**Options**:
- false - Normal logging
- true - Verbose debug logging

**When to change**:
- Enable when troubleshooting issues
- Keep disabled for normal operation

## Example Scenarios

### Scenario 1: High Quality, Long Recordings
```
max_duration: 120 minutes
resolution: 9999 (source)
framerate: 60
interval: 1
debug: false
```
**Use case**: Recording important streams in full quality with 2-hour chunks

### Scenario 2: Space-Saving Mode
```
max_duration: 30 minutes
resolution: 720
framerate: 30
interval: 2
debug: false
```
**Use case**: Saving disk space and bandwidth with smaller files

### Scenario 3: Quick Testing
```
max_duration: 15 minutes
resolution: 480
framerate: 30
interval: 1
debug: true
```
**Use case**: Testing the workflow with small files and debug output

### Scenario 4: Default (Recommended)
```
max_duration: 45 minutes
resolution: 9999
framerate: 60
interval: 1
debug: false
```
**Use case**: Balanced settings for most use cases

## Important Notes

### Artifact Size Limits
- GitHub Actions has a **10GB per artifact** limit
- Smaller `max_duration` = smaller files = safer
- 45 minutes is recommended to stay well under limits

### File Size Estimates
Approximate file sizes for 45-minute chunks:

| Resolution | Framerate | Estimated Size |
|------------|-----------|----------------|
| 1080p      | 60fps     | ~2-3GB         |
| 1080p      | 30fps     | ~1-2GB         |
| 720p       | 60fps     | ~1-2GB         |
| 720p       | 30fps     | ~500MB-1GB     |
| 480p       | 30fps     | ~300-500MB     |

### Recording Duration vs Job Timeout
- The workflow has a **5-hour timeout** for recording
- After 5 hours, it auto-restarts to continue recording
- `max_duration` only affects individual file sizes, not total recording time

### When Parameters Don't Apply
- Parameters only work when **manually triggering** the workflow
- Scheduled runs (cron) and automatic restarts use **default values**
- To change defaults permanently, edit the workflow file

## Troubleshooting

### "Artifact size approaching 10GB limit"
**Solution**: Reduce `max_duration` to 30 or 15 minutes

### "Recording files are too small"
**Solution**: Increase `max_duration` to 60-120 minutes

### "Stream detection is slow"
**Solution**: Set `interval` to 1 minute

### "Files are too large to upload"
**Solution**: Reduce `resolution` to 720p or lower

### "Need to debug issues"
**Solution**: Enable `debug: true` for verbose logging

## API Reference

The workflow inputs map directly to goondvr CLI flags:

| Workflow Input | CLI Flag         | Description                    |
|----------------|------------------|--------------------------------|
| max_duration   | -max-duration    | Minutes per file               |
| resolution     | -resolution      | Target resolution              |
| framerate      | -framerate       | Target framerate               |
| interval       | -interval        | Check interval (minutes)       |
| debug          | --debug          | Enable debug logging           |

## See Also

- [Edge Cases Fixed](EDGE_CASES_FIXED.md)
- [Bug Fixes](BUGS_FOUND.md)
- [All Fixes Complete](ALL_FIXES_COMPLETE.md)
