package jobs

import (
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

// TestJobCancellation tests cancellation functionality
func TestJobCancellation(t *testing.T) {
	t.Run("Cancel running job", func(t *testing.T) {
		job := NewJob("test_job", fiber.Map{})
		job.Start()

		err := CancelJob(job.ID)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if job.Status != StatusCancelled {
			t.Errorf("Expected status cancelled, got %s", job.Status)
		}

		if !job.IsCancelled() {
			t.Error("IsCancelled should return true")
		}
	})

	t.Run("Cancel non-existent job", func(t *testing.T) {
		err := CancelJob("non-existent-id")
		if err == nil {
			t.Error("Expected error for non-existent job")
		}
	})

	t.Run("Cancel completed job does nothing", func(t *testing.T) {
		job := NewJob("test_job", fiber.Map{})
		job.Start()
		job.Complete(fiber.Map{})

		// Try to cancel completed job
		job.Cancel()

		// Status should still be completed, not cancelled
		if job.Status != StatusCompleted {
			t.Errorf("Expected status completed, got %s", job.Status)
		}
	})

	t.Run("IsCancelled on non-cancelled job", func(t *testing.T) {
		job := NewJob("test_job", fiber.Map{})
		job.Start()

		if job.IsCancelled() {
			t.Error("IsCancelled should return false for non-cancelled job")
		}
	})
}

// TestJobStatuses tests all job status transitions
func TestJobStatuses(t *testing.T) {
	t.Run("Pending to Running", func(t *testing.T) {
		job := NewJob("test_job", fiber.Map{})

		if job.Status != StatusPending {
			t.Errorf("New job should be pending, got %s", job.Status)
		}

		job.Start()

		if job.Status != StatusRunning {
			t.Errorf("Started job should be running, got %s", job.Status)
		}

		if job.StartedAt == nil {
			t.Error("StartedAt should be set after Start()")
		}
	})

	t.Run("Running to Completed", func(t *testing.T) {
		job := NewJob("test_job", fiber.Map{})
		job.Start()

		job.Complete(fiber.Map{"result": "success"})

		if job.Status != StatusCompleted {
			t.Errorf("Expected completed status, got %s", job.Status)
		}

		if job.EndedAt == nil {
			t.Error("EndedAt should be set")
		}

		if job.Progress != 100 {
			t.Errorf("Progress should be 100, got %d", job.Progress)
		}
	})

	t.Run("Running to Failed", func(t *testing.T) {
		job := NewJob("test_job", fiber.Map{})
		job.Start()

		job.Fail("Test error")

		if job.Status != StatusFailed {
			t.Errorf("Expected failed status, got %s", job.Status)
		}

		if job.Error != "Test error" {
			t.Errorf("Expected error message, got %s", job.Error)
		}

		if job.EndedAt == nil {
			t.Error("EndedAt should be set")
		}
	})
}

// TestJobConcurrency tests concurrent job operations
func TestJobConcurrency(t *testing.T) {
	t.Run("Concurrent job creation", func(t *testing.T) {
		ClearAllJobs()

		var wg sync.WaitGroup
		jobCount := 100

		for i := 0; i < jobCount; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_ = NewJob("concurrent_job", fiber.Map{"index": i})
			}(i)
		}

		wg.Wait()

		allJobs := GetAllJobs()
		if len(allJobs) != jobCount {
			t.Errorf("Expected %d jobs, got %d", jobCount, len(allJobs))
		}
	})

	t.Run("Concurrent progress updates", func(t *testing.T) {
		job := NewJob("test_job", fiber.Map{})
		job.Start()

		var wg sync.WaitGroup
		updateCount := 50

		for i := 0; i < updateCount; i++ {
			wg.Add(1)
			go func(progress int) {
				defer wg.Done()
				job.UpdateProgress(progress)
			}(i)
		}

		wg.Wait()

		// Job should still be accessible and not corrupted
		if job.Status != StatusRunning {
			t.Error("Job status should still be running after concurrent updates")
		}
	})

	t.Run("Concurrent message updates", func(t *testing.T) {
		job := NewJob("test_job", fiber.Map{})
		job.Start()

		var wg sync.WaitGroup
		messageCount := 50

		for i := 0; i < messageCount; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				job.UpdateMessage("Message " + string(rune(i)))
			}(i)
		}

		wg.Wait()

		// Job should still be accessible
		jobMap := job.ToMap()
		if jobMap["status"] != StatusRunning {
			t.Error("Job should still be running")
		}
	})
}

// TestJobLifecycle tests complete job lifecycle
func TestJobLifecycle(t *testing.T) {
	job := NewJob("lifecycle_job", fiber.Map{
		"initial": "data",
	})

	// Check initial state
	if job.Status != StatusPending {
		t.Error("Job should start as pending")
	}

	// Start job
	job.Start()
	if job.Status != StatusRunning {
		t.Error("Job should be running after Start()")
	}

	// Update progress multiple times
	job.UpdateProgress(25)
	if job.Progress != 25 {
		t.Error("Progress should be 25")
	}

	job.UpdateProgressWithMessage(50, "Halfway there")
	if job.Progress != 50 || job.Message != "Halfway there" {
		t.Error("Progress and message should update")
	}

	job.UpdateProgress(75)
	if job.Progress != 75 {
		t.Error("Progress should be 75")
	}

	// Complete job
	job.Complete(fiber.Map{
		"final": "result",
	})

	if job.Status != StatusCompleted {
		t.Error("Job should be completed")
	}

	if job.Progress != 100 {
		t.Error("Progress should be 100 when completed")
	}

	// Verify ToMap includes all data
	jobMap := job.ToMap()
	if jobMap["id"] == nil || jobMap["type"] == nil {
		t.Error("ToMap should include id and type")
	}

	if input, ok := jobMap["input"].(fiber.Map); !ok || input["initial"] != "data" {
		t.Error("ToMap should preserve input")
	}

	if result, ok := jobMap["result"].(fiber.Map); !ok || result["final"] != "result" {
		t.Error("ToMap should include result")
	}
}

// TestJobTimestamps tests timestamp tracking
func TestJobTimestamps(t *testing.T) {
	t.Run("CreatedAt is set on creation", func(t *testing.T) {
		before := time.Now()
		job := NewJob("test_job", fiber.Map{})
		after := time.Now()

		if job.CreatedAt.Before(before) || job.CreatedAt.After(after) {
			t.Error("CreatedAt should be set during job creation")
		}
	})

	t.Run("UpdatedAt changes on updates", func(t *testing.T) {
		job := NewJob("test_job", fiber.Map{})
		job.Start()
		originalUpdatedAt := job.UpdatedAt

		time.Sleep(10 * time.Millisecond)

		job.UpdateProgress(50)

		if !job.UpdatedAt.After(originalUpdatedAt) {
			t.Error("UpdatedAt should change after update")
		}
	})

	t.Run("StartedAt is set on Start", func(t *testing.T) {
		job := NewJob("test_job", fiber.Map{})

		if job.StartedAt != nil {
			t.Error("StartedAt should be nil before Start()")
		}

		job.Start()

		if job.StartedAt == nil {
			t.Error("StartedAt should be set after Start()")
		}
	})

	t.Run("EndedAt is set on completion", func(t *testing.T) {
		job := NewJob("test_job", fiber.Map{})
		job.Start()

		if job.EndedAt != nil {
			t.Error("EndedAt should be nil before completion")
		}

		job.Complete(fiber.Map{})

		if job.EndedAt == nil {
			t.Error("EndedAt should be set after Complete()")
		}
	})

	t.Run("EndedAt is set on failure", func(t *testing.T) {
		job := NewJob("test_job", fiber.Map{})
		job.Start()

		job.Fail("error")

		if job.EndedAt == nil {
			t.Error("EndedAt should be set after Fail()")
		}
	})
}

// TestCleanupOldJobsEdgeCases tests cleanup edge cases
func TestCleanupOldJobsEdgeCases(t *testing.T) {
	t.Run("Cleanup only affects old completed/failed jobs", func(t *testing.T) {
		ClearAllJobs()

		// Old failed job - should be cleaned
		oldFailed := NewJob("old_failed", fiber.Map{})
		oldFailed.Start()
		oldFailed.Fail("error")
		oldFailed.UpdatedAt = time.Now().Add(-25 * time.Hour)

		// Old completed job - should be cleaned
		oldCompleted := NewJob("old_completed", fiber.Map{})
		oldCompleted.Start()
		oldCompleted.Complete(fiber.Map{})
		oldCompleted.UpdatedAt = time.Now().Add(-25 * time.Hour)

		// Old running job - should NOT be cleaned
		oldRunning := NewJob("old_running", fiber.Map{})
		oldRunning.Start()
		oldRunning.UpdatedAt = time.Now().Add(-25 * time.Hour)

		// Old pending job - should NOT be cleaned
		oldPending := NewJob("old_pending", fiber.Map{})
		oldPending.UpdatedAt = time.Now().Add(-25 * time.Hour)

		// Recent completed job - should NOT be cleaned
		recentCompleted := NewJob("recent_completed", fiber.Map{})
		recentCompleted.Start()
		recentCompleted.Complete(fiber.Map{})

		CleanupOldJobs()

		if GetJob(oldFailed.ID) != nil {
			t.Error("Old failed job should be cleaned")
		}

		if GetJob(oldCompleted.ID) != nil {
			t.Error("Old completed job should be cleaned")
		}

		if GetJob(oldRunning.ID) == nil {
			t.Error("Old running job should NOT be cleaned")
		}

		if GetJob(oldPending.ID) == nil {
			t.Error("Old pending job should NOT be cleaned")
		}

		if GetJob(recentCompleted.ID) == nil {
			t.Error("Recent completed job should NOT be cleaned")
		}
	})
}

// TestGetJobNotFound tests getting non-existent job
func TestGetJobNotFound(t *testing.T) {
	job := GetJob("does-not-exist")
	if job != nil {
		t.Error("GetJob should return nil for non-existent job")
	}
}

// TestClearAllJobs tests clearing all jobs
func TestClearAllJobs(t *testing.T) {
	// Create some jobs
	NewJob("job1", fiber.Map{})
	NewJob("job2", fiber.Map{})
	NewJob("job3", fiber.Map{})

	if len(GetAllJobs()) == 0 {
		t.Error("Should have jobs before clear")
	}

	ClearAllJobs()

	if len(GetAllJobs()) != 0 {
		t.Error("Should have no jobs after clear")
	}
}

// TestJobInputAndResult tests input and result handling
func TestJobInputAndResult(t *testing.T) {
	t.Run("Complex input data", func(t *testing.T) {
		complexInput := fiber.Map{
			"string":  "value",
			"number":  42,
			"boolean": true,
			"nested": fiber.Map{
				"key": "nested_value",
			},
			"array": []string{"a", "b", "c"},
		}

		job := NewJob("test_job", complexInput)

		jobMap := job.ToMap()
		if input, ok := jobMap["input"].(fiber.Map); !ok {
			t.Error("Input should be preserved as fiber.Map")
		} else {
			if input["string"] != "value" || input["number"] != 42 || input["boolean"] != true {
				t.Error("Input values should be preserved")
			}
		}
	})

	t.Run("Complex result data", func(t *testing.T) {
		job := NewJob("test_job", fiber.Map{})
		job.Start()

		complexResult := fiber.Map{
			"files_processed": 100,
			"errors":          0,
			"duration":        "5s",
			"details": fiber.Map{
				"success": 100,
				"failed":  0,
			},
		}

		job.Complete(complexResult)

		jobMap := job.ToMap()
		if result, ok := jobMap["result"].(fiber.Map); !ok {
			t.Error("Result should be preserved as fiber.Map")
		} else {
			if result["files_processed"] != 100 {
				t.Error("Result values should be preserved")
			}
		}
	})
}
