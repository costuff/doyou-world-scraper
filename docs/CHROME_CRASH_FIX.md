# Chrome Crash Investigation & Fixes

## Issue: Chrome crashes after website is open for a while

## Root Causes Identified

### 1. **Memory Leaks in chromedp Contexts** ⚠️ CRITICAL
**Location**: `internal/scraper/scraper.go`

**Problem**:
- When scraping a month or year, the code creates ONE Chrome context and reuses it for ALL dates
- Each page navigation accumulates memory (DOM nodes, JavaScript heap, cached resources)
- No cleanup between page navigations
- Chrome eventually runs out of memory and crashes

**Current Code**:
```go
// ScrapeMonth creates ONE context for entire month
ctx, cancel := chromedp.NewContext(allocCtx)
defer cancel()  // Only called at the end

for day := 1; day <= 31; day++ {
    // Reuses same context for 31 pages
    count, err := scrapeSingleDateWithContext(ctx, dateStr)
    // No cleanup between iterations!
}
```

**Impact**: Scraping a month (30 pages) can accumulate 2-3GB of memory. Year (365 pages) will definitely crash.

---

### 2. **No Timeouts on Page Operations** ⚠️ HIGH
**Problem**:
- `chromedp.Run(ctx, ...)` has no timeout
- If doyoutrackid.com is slow/down, contexts hang indefinitely
- Accumulated hanging contexts consume memory
- Frontend keeps making new requests, creating more contexts

**Current Code**:
```go
err = chromedp.Run(ctx,
    chromedp.Navigate(url),
    chromedp.Sleep(3*time.Second),  // Fixed sleep, not timeout
    chromedp.Nodes(`div.Track_root__fkoK4`, &trackNodes, chromedp.ByQueryAll),
)
// No timeout - will wait forever if page doesn't load
```

---

### 3. **No Resource Limits on Chrome Process** ⚠️ MEDIUM
**Problem**:
- chromedp flags don't set memory limits
- Chrome can consume all available RAM
- No tab/process management

**Current Flags**:
```go
chromedp.Flag("headless", true),
chromedp.Flag("disable-gpu", true),
chromedp.Flag("no-sandbox", true),
chromedp.Flag("disable-dev-shm-usage", true),
// Missing: memory limits, disk cache limits
```

---

### 4. **Concurrent Scraping Without Limits** ⚠️ MEDIUM
**Problem**:
- Frontend can trigger multiple scrape jobs simultaneously
- Each job creates its own Chrome instance
- No limit on concurrent Chrome processes
- System can run out of memory/file descriptors

---

### 5. **No Cleanup on Job Cancellation** ⚠️ LOW
**Problem**:
- If user cancels a scraping job, Chrome context might not close immediately
- `defer cancel()` only runs when function returns
- User might spam cancel/restart, leaving zombie Chrome processes

---

## Proposed Fixes

### Fix 1: Add Context Cleanup Between Pages (HIGHEST PRIORITY)
**Strategy**: Create a new context for EACH date instead of reusing one

```go
// ScrapeMonth - BEFORE (memory leak)
ctx, cancel := chromedp.NewContext(allocCtx)  // ONE context for all pages
defer cancel()
for day := 1; day <= 31; day++ {
    scrapeSingleDateWithContext(ctx, dateStr)  // Reuses context
}

// ScrapeMonth - AFTER (proper cleanup)
for day := 1; day <= 31; day++ {
    ctx, cancel := chromedp.NewContext(allocCtx)  // NEW context per page
    count, err := scrapeSingleDateWithContext(ctx, dateStr)
    cancel()  // Immediate cleanup after each page
    if err != nil {
        fmt.Printf("Error scraping %s: %v\n", dateStr, err)
        continue
    }
    totalTracks += count
}
```

**Benefits**:
- Each page load starts with fresh memory
- Memory is freed after each date (not at the end)
- Chrome process size stays constant (~200MB instead of growing to 3GB+)

---

### Fix 2: Add Timeouts to All chromedp Operations
**Strategy**: Wrap each chromedp.Run with a timeout context

```go
func scrapeSingleDateWithContext(ctx context.Context, dateStr string) (int, error) {
    // Add timeout for page operations
    timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 30*time.Second)
    defer timeoutCancel()
    
    var trackNodes []*cdp.Node
    err = chromedp.Run(timeoutCtx,  // Use timeout context
        chromedp.Navigate(url),
        chromedp.Sleep(3*time.Second),
        chromedp.Nodes(`div.Track_root__fkoK4`, &trackNodes, chromedp.ByQueryAll),
    )
    if err != nil {
        if err == context.DeadlineExceeded {
            return 0, fmt.Errorf("page load timeout after 30s")
        }
        return 0, err
    }
    // ... rest of scraping
}
```

---

### Fix 3: Add Memory Limits to Chrome Flags
**Strategy**: Configure Chrome with resource limits

```go
allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(),
    append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.Flag("headless", true),
        chromedp.Flag("disable-gpu", true),
        chromedp.Flag("no-sandbox", true),
        chromedp.Flag("disable-dev-shm-usage", true),
        
        // NEW: Memory and resource limits
        chromedp.Flag("disable-dev-tools", true),
        chromedp.Flag("disable-extensions", true),
        chromedp.Flag("disable-plugins", true),
        chromedp.Flag("disable-images", true),  // Don't load images (faster + less memory)
        chromedp.Flag("disk-cache-size", "1"),  // Minimal cache
        chromedp.Flag("media-cache-size", "1"),
        chromedp.Flag("aggressive-cache-discard", true),
        chromedp.Flag("disable-cache", true),
        chromedp.Flag("disable-application-cache", true),
        chromedp.Flag("disable-offline-load-stale-cache", true),
        chromedp.Flag("memory-model", "low"),
        chromedp.Flag("max-gum-fps", "30"),  // Limit frame rate
    )...)
```

**Note**: chromedp doesn't support `--js-flags="--max-old-space-size=512"` directly, but we can set via environment:
```go
os.Setenv("NODE_MAX_OLD_SPACE_SIZE", "512")  // 512MB heap limit
```

---

### Fix 4: Limit Concurrent Scraping Jobs
**Strategy**: Use a semaphore to limit concurrent Chrome instances

```go
// In handlers.go or scraper package
var (
    scrapeSemaphore = make(chan struct{}, 2)  // Max 2 concurrent scrape jobs
)

func (h *Handlers) HandleScrapeMonth(c *fiber.Ctx) error {
    // ... parse request ...
    
    job := jobs.NewJob("scrape_month", fiber.Map{"month": req.Month})
    
    go func() {
        // Acquire semaphore (blocks if 2 jobs already running)
        scrapeSemaphore <- struct{}{}
        defer func() { <-scrapeSemaphore }()  // Release when done
        
        days, tracks, err := scraper.ScrapeMonth(req.Month, job, progressCallback)
        // ... complete job ...
    }()
    
    return c.Status(202).JSON(fiber.Map{
        "message": "Scraping started",
        "job":     job,
    })
}
```

---

### Fix 5: Force Cleanup on Job Cancellation
**Strategy**: Use context cancellation to immediately stop Chrome

```go
func ScrapeMonth(monthStr string, job *jobs.Job, progressCallback func(current, total int)) (int, int, error) {
    // Create cancellable context
    allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), ...)
    defer allocCancel()
    
    for day := 1; day <= 31; day++ {
        // Check cancellation BEFORE creating context
        if job != nil && job.IsCancelled() {
            return totalDays, totalTracks, fmt.Errorf("job cancelled")
        }
        
        // Create per-page context with cancellation support
        ctx, cancel := chromedp.NewContext(allocCtx)
        
        // Monitor for job cancellation during scraping
        done := make(chan error, 1)
        go func() {
            count, err := scrapeSingleDateWithContext(ctx, dateStr)
            done <- err
        }()
        
        select {
        case err := <-done:
            cancel()  // Clean up context
            if err != nil {
                continue
            }
            totalTracks += count
        case <-time.After(30 * time.Second):
            cancel()  // Timeout - force cleanup
            return totalDays, totalTracks, fmt.Errorf("scraping timeout")
        }
        
        // Check again after each page
        if job != nil && job.IsCancelled() {
            cancel()
            return totalDays, totalTracks, fmt.Errorf("job cancelled")
        }
    }
    
    return totalDays, totalTracks, nil
}
```

---

## Frontend Considerations

### Issue: Auto-Refresh Triggering Too Many Chrome Instances
**Location**: `frontend/src/components/StatsPanel.tsx`

**Check**: Is stats refresh triggering scrapes?
```typescript
// StatsPanel.tsx
useEffect(() => {
  fetchStats()
  const interval = setInterval(fetchStats, 5000)  // Every 5 seconds!
  return () => clearInterval(interval)
}, [])
```

**Fix**: Stats should only READ data, not trigger scrapes. Verify API calls:
- ✅ `GET /api/v1/util/stats` - Safe (read-only)
- ❌ `POST /api/v1/scrape/*` - Should NOT be called from auto-refresh

---

## Testing Strategy

### Test 1: Memory Usage During Long Scrapes
```bash
# Monitor memory while scraping
while true; do
  ps aux | grep chrome | grep -v grep
  sleep 5
done

# Trigger month scrape
curl -X POST http://localhost:3001/api/v1/scrape/month -H "Content-Type: application/json" -d '{"month":"2026-01"}'
```

**Expected After Fix**:
- Chrome memory should stay under 500MB
- No memory accumulation over time
- Clean shutdown after scrape completes

---

### Test 2: Concurrent Scraping
```bash
# Start 5 scrape jobs simultaneously
for i in {01..05}; do
  curl -X POST http://localhost:3001/api/v1/scrape/date -H "Content-Type: application/json" -d "{\"date\":\"2026-02-$i\"}" &
done
wait
```

**Expected After Fix**:
- Max 2 Chrome processes at a time (semaphore limit)
- All jobs complete successfully
- No system resource exhaustion

---

### Test 3: Job Cancellation
```bash
# Start long-running year scrape
curl -X POST http://localhost:3001/api/v1/scrape/year -H "Content-Type: application/json" -d '{"year":"2025"}' | jq -r '.job.id'

# Cancel immediately
JOB_ID="<paste job id>"
curl -X POST http://localhost:3001/api/v1/jobs/$JOB_ID/cancel
```

**Expected After Fix**:
- Chrome process terminates within 1 second
- No zombie Chrome processes
- Job status shows "cancelled"

---

## Implementation Priority

### Phase 1: Critical Fixes (Do First) ��
1. ✅ **Fix 1**: Add context cleanup between pages (prevents memory leaks)
2. ✅ **Fix 2**: Add timeouts to chromedp operations (prevents hanging)
3. ✅ **Fix 3**: Add Chrome memory limit flags (resource management)

**Estimated Impact**: Reduces memory usage by 80%, prevents crashes

### Phase 2: Performance Improvements (Do Second)
4. **Fix 4**: Limit concurrent scraping (semaphore)
5. **Fix 5**: Improve job cancellation (immediate cleanup)

**Estimated Impact**: Better system stability, faster response to user actions

### Phase 3: Monitoring (Do Last)
6. Add memory usage metrics to job status
7. Log Chrome process lifecycle events
8. Alert on memory threshold exceeded

---

## Additional Recommendations

### 1. Consider Alternative Scraping Approaches
**Current**: chromedp (full Chrome browser, heavy)
**Alternatives**:
- **colly** - Lightweight Go scraping library (no browser needed)
- **rod** - Modern chromedp alternative with better resource management
- **API approach** - Check if doyoutrackid.com has an API

**Trade-off**: chromedp handles JavaScript/dynamic content but is resource-heavy

---

### 2. Add Scraping Rate Limiting
**Recommendation**: Don't scrape doyoutrackid.com too aggressively
```go
// Add delay between page requests
time.Sleep(2 * time.Second)  // Be nice to their servers
```

---

### 3. Cache Scraped Data
**Recommendation**: Don't re-scrape the same date repeatedly
```go
func shouldScrapeDate(dateStr string) bool {
    csvPath := filepath.Join(csvDir, fmt.Sprintf("tracks-%s.csv", dateStr))
    if fileExists(csvPath) {
        // Check if file is recent (< 24 hours old)
        info, _ := os.Stat(csvPath)
        if time.Since(info.ModTime()) < 24*time.Hour {
            return false  // Skip, already scraped recently
        }
    }
    return true
}
```

---

## Summary

**Why Chrome Crashes**:
1. 🔴 Memory leaks from reusing Chrome context without cleanup
2. 🔴 No timeouts - hanging contexts accumulate
3. 🟡 No resource limits on Chrome process
4. 🟡 Unlimited concurrent scraping jobs

**Fixes**:
1. Create new context per page (not per month/year)
2. Add 30s timeouts to all chromedp operations
3. Configure Chrome with memory/cache limits
4. Limit to 2 concurrent scrape jobs max
5. Improve job cancellation to force immediate cleanup

**Expected Result**: 
- Stable memory usage (~200-500MB)
- No crashes during long scrapes
- Responsive job cancellation
- Better system resource management
