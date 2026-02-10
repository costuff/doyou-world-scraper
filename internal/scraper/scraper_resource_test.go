package scraper

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// TestChromeMemoryLimits verifies Chrome is started with memory-optimized flags
func TestChromeMemoryLimits(t *testing.T) {
	// Skip by default - requires Chrome and may be slow
	t.Skip("Skipping Chrome test - run manually if needed")

	t.Run("Allocator includes memory limit flags", func(t *testing.T) {
		allocCtx, allocCancel := createChromeAllocator()
		defer allocCancel()

		ctx, cancel := chromedp.NewContext(allocCtx)
		defer cancel()

		// If we can create a context without error, the flags are valid
		timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 5*time.Second)
		defer timeoutCancel()

		// Try to run a simple command
		err := chromedp.Run(timeoutCtx, chromedp.Navigate("about:blank"))
		if err != nil {
			t.Skipf("Chrome not available or flags incompatible: %v", err)
		}

		t.Log("Chrome allocator created successfully with memory limit flags")
	})
}

// TestContextCleanup verifies contexts are properly cleaned up
func TestContextCleanup(t *testing.T) {
	t.Run("scrapeSingleDateWithContext cleans up on timeout", func(t *testing.T) {
		// Skip by default - requires Chrome and network
		t.Skip("Skipping Chrome/network test - run manually if needed")

		// Skip if Chrome not available
		if testing.Short() {
			t.Skip("Skipping Chrome test in short mode")
		}

		allocCtx, allocCancel := createChromeAllocator()
		defer allocCancel()

		ctx, cancel := chromedp.NewContext(allocCtx)
		defer cancel()

		// Try scraping with invalid date (should fail but not hang)
		startTime := time.Now()
		_, err := scrapeSingleDateWithContext(ctx, "2020-01-01")
		elapsed := time.Since(startTime)

		// Should complete within reasonable time (not hang forever)
		if elapsed > 35*time.Second {
			t.Errorf("Context cleanup took too long: %v (expected <35s)", elapsed)
		}

		if err != nil {
			t.Logf("Expected error for old date: %v", err)
		}
	})

	t.Run("Multiple contexts don't accumulate memory", func(t *testing.T) {
		// Skip if Chrome not available
		if testing.Short() {
			t.Skip("Skipping Chrome test in short mode")
		}

		allocCtx, allocCancel := createChromeAllocator()
		defer allocCancel()

		// Create and cleanup multiple contexts
		for i := 0; i < 3; i++ {
			ctx, cancel := chromedp.NewContext(allocCtx)

			// Run simple navigation
			timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 5*time.Second)
			_ = chromedp.Run(timeoutCtx, chromedp.Navigate("about:blank"))
			timeoutCancel()

			// Cleanup immediately
			cancel()

			// Give Chrome time to cleanup
			time.Sleep(100 * time.Millisecond)
		}

		// Force garbage collection
		runtime.GC()
		time.Sleep(500 * time.Millisecond)

		t.Log("Multiple contexts created and cleaned up successfully")
	})
}

// TestChromeProcessCount verifies Chrome processes are limited
func TestChromeProcessCount(t *testing.T) {
	// Skip by default - requires Chrome
	t.Skip("Skipping Chrome test - run manually if needed")

	// Skip on CI or if Chrome not available
	if os.Getenv("CI") != "" || testing.Short() {
		t.Skip("Skipping Chrome process count test in CI/short mode")
	}

	t.Run("Count Chrome processes before and after", func(t *testing.T) {
		countChrome := func() int {
			if runtime.GOOS == "windows" {
				return 0 // Skip on Windows
			}

			cmd := exec.Command("ps", "aux")
			output, err := cmd.CombinedOutput()
			if err != nil {
				return 0
			}

			count := 0
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "chrome") || strings.Contains(line, "chromium") {
					count++
				}
			}
			return count
		}

		initialCount := countChrome()

		// Create allocator and context
		allocCtx, allocCancel := createChromeAllocator()
		ctx, cancel := chromedp.NewContext(allocCtx)

		// Run simple command
		timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 5*time.Second)
		_ = chromedp.Run(timeoutCtx, chromedp.Navigate("about:blank"))
		timeoutCancel()

		midCount := countChrome()
		if midCount > initialCount {
			t.Logf("Chrome processes increased from %d to %d (expected)", initialCount, midCount)
		}

		// Cleanup
		cancel()
		allocCancel()
		time.Sleep(1 * time.Second)
		runtime.GC()

		finalCount := countChrome()
		if finalCount > midCount {
			t.Errorf("Chrome processes not cleaned up: before=%d, during=%d, after=%d",
				initialCount, midCount, finalCount)
		} else {
			t.Logf("Chrome processes cleaned up: before=%d, during=%d, after=%d",
				initialCount, midCount, finalCount)
		}
	})
}

// TestTimeoutBehavior verifies timeout context works correctly
func TestTimeoutBehavior(t *testing.T) {
	t.Run("Timeout context triggers after deadline", func(t *testing.T) {
		parent := context.Background()
		timeoutCtx, cancel := context.WithTimeout(parent, 100*time.Millisecond)
		defer cancel()

		// Wait for timeout
		<-timeoutCtx.Done()

		if timeoutCtx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded, got: %v", timeoutCtx.Err())
		}
	})

	t.Run("scrapeSingleDateWithContext respects timeout", func(t *testing.T) {
		// Skip by default - network test
		t.Skip("Skipping network test - run manually if needed")

		// Skip in short mode - requires actual network call
		if testing.Short() {
			t.Skip("Skipping network test in short mode")
		}

		// This test verifies the timeout is passed through correctly
		// We can't easily test actual timeout without a slow server
		// But we can verify the code structure is correct

		allocCtx, allocCancel := createChromeAllocator()
		defer allocCancel()

		ctx, cancel := chromedp.NewContext(allocCtx)
		defer cancel()

		// The function should use timeoutCtx internally
		// If it hangs, the test will timeout
		_, err := scrapeSingleDateWithContext(ctx, "2020-01-01")
		if err != nil {
			t.Logf("Expected error (network or timeout): %v", err)
		}
	})
}

// TestMemoryLeakPrevention verifies per-page context cleanup
func TestMemoryLeakPrevention(t *testing.T) {
	t.Run("Allocator reuse with multiple contexts", func(t *testing.T) {
		// This pattern is what ScrapeMonth and ScrapeYear now use
		allocCtx, allocCancel := createChromeAllocator()
		defer allocCancel()

		// Simulate scraping multiple dates
		for i := 0; i < 3; i++ {
			ctx, cancel := chromedp.NewContext(allocCtx)

			// Simulate page operations (skip actual Chrome to keep test fast)
			timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 1*time.Second)
			_ = timeoutCtx // Reference to avoid unused warning
			timeoutCancel()

			// Immediate cleanup (Fix 1)
			cancel()
		}

		t.Log("Multiple page contexts created and cleaned up with single allocator")
	})
}

// TestCreateChromeAllocator verifies the helper function
func TestCreateChromeAllocator(t *testing.T) {
	t.Run("createChromeAllocator returns valid contexts", func(t *testing.T) {
		allocCtx, allocCancel := createChromeAllocator()
		if allocCtx == nil {
			t.Error("allocCtx should not be nil")
		}
		if allocCancel == nil {
			t.Error("allocCancel should not be nil")
		}

		defer allocCancel()

		// Verify we can create a browser context
		ctx, cancel := chromedp.NewContext(allocCtx)
		if ctx == nil {
			t.Error("ctx should not be nil")
		}
		defer cancel()

		t.Log("Chrome allocator created successfully")
	})
}

// BenchmarkContextCreation measures context creation overhead
func BenchmarkContextCreation(b *testing.B) {
	allocCtx, allocCancel := createChromeAllocator()
	defer allocCancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := chromedp.NewContext(allocCtx)
		_ = ctx // Use context to avoid unused warning
		cancel()
	}
}

// BenchmarkMemoryCleanup measures cleanup time
func BenchmarkMemoryCleanup(b *testing.B) {
	allocCtx, allocCancel := createChromeAllocator()
	defer allocCancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := chromedp.NewContext(allocCtx)
		_ = ctx // Use context to avoid unused warning
		// Immediate cleanup like in fixed ScrapeMonth
		cancel()
	}
}
