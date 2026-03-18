package jobs

import (
	"testing"
	"time"

	fiber_websocket "github.com/gofiber/websocket/v2"

	"github.com/gofiber/fiber/v2"
)

func TestNewJob(t *testing.T) {
	job := NewJob("test_job", fiber.Map{
		"key": "value",
	})

	if job.ID == "" {
		t.Error("Job ID should not be empty")
	}

	if job.Type != "test_job" {
		t.Errorf("Expected job type 'test_job', got '%s'", job.Type)
	}

	// Jobs should start as pending (Phase 2: Chrome semaphore fix)
	if job.Status != StatusPending {
		t.Errorf("Expected status 'pending', got '%s'", job.Status)
	}

	if job.Progress != 0 {
		t.Errorf("Expected progress 0, got %d", job.Progress)
	}

	// StartedAt should NOT be set until Start() is called
	if job.StartedAt != nil {
		t.Error("StartedAt should not be set before Start() is called")
	}

	// Now start the job and verify it transitions to running
	job.Start()

	if job.Status != StatusRunning {
		t.Errorf("Expected status 'running' after Start(), got '%s'", job.Status)
	}

	if job.StartedAt == nil {
		t.Error("StartedAt should be set after Start()")
	}
}

func TestJobUpdateProgress(t *testing.T) {
	job := NewJob("test_job", fiber.Map{})
	job.Start() // Must start job before updating progress

	job.UpdateProgress(50)

	if job.Progress != 50 {
		t.Errorf("Expected progress 50, got %d", job.Progress)
	}

	if job.Status != StatusRunning {
		t.Errorf("Expected status 'running', got '%s'", job.Status)
	}
}

func TestJobUpdateMessage(t *testing.T) {
	job := NewJob("test_download", fiber.Map{})
	job.Start() // Must start job before updating message

	message := "Downloading: Track Name by Artist"
	job.UpdateMessage(message)

	if job.Message != message {
		t.Errorf("Expected message '%s', got '%s'", message, job.Message)
	}

	if job.Status != StatusRunning {
		t.Errorf("Expected status 'running', got '%s'", job.Status)
	}
}

func TestJobUpdateProgressWithMessage(t *testing.T) {
	job := NewJob("test_download", fiber.Map{})

	message := "Downloading: Another Track by Artist"
	job.UpdateProgressWithMessage(75, message)

	if job.Progress != 75 {
		t.Errorf("Expected progress 75, got %d", job.Progress)
	}

	if job.Message != message {
		t.Errorf("Expected message '%s', got '%s'", message, job.Message)
	}

	// Check message is included in ToMap
	jobMap := job.ToMap()
	if jobMap["message"] != message {
		t.Errorf("Expected message in ToMap '%s', got '%v'", message, jobMap["message"])
	}
}

func TestJobComplete(t *testing.T) {
	job := NewJob("test_job", fiber.Map{})

	result := fiber.Map{
		"files_processed": 100,
	}
	job.Complete(result)

	if job.Status != StatusCompleted {
		t.Errorf("Expected status 'completed', got '%s'", job.Status)
	}

	if job.Progress != 100 {
		t.Errorf("Expected progress 100, got %d", job.Progress)
	}

	if job.EndedAt == nil {
		t.Error("EndedAt should be set")
	}

	if job.Result == nil {
		t.Error("Result should be set")
	}

	if val, ok := job.Result["files_processed"]; !ok || val != 100 {
		t.Error("Result should contain files_processed=100")
	}
}

func TestJobFail(t *testing.T) {
	job := NewJob("test_job", fiber.Map{})

	errorMsg := "Something went wrong"
	job.Fail(errorMsg)

	if job.Status != StatusFailed {
		t.Errorf("Expected status 'failed', got '%s'", job.Status)
	}

	if job.Error != errorMsg {
		t.Errorf("Expected error '%s', got '%s'", errorMsg, job.Error)
	}

	if job.EndedAt == nil {
		t.Error("EndedAt should be set")
	}
}

func TestGetJob(t *testing.T) {
	job := NewJob("test_job", fiber.Map{})

	retrieved := GetJob(job.ID)

	if retrieved == nil {
		t.Fatal("Job should be retrievable")
	}

	if retrieved.ID != job.ID {
		t.Errorf("Expected job ID '%s', got '%s'", job.ID, retrieved.ID)
	}
}

func TestGetAllJobs(t *testing.T) {
	// Clear existing jobs for clean test
	jobsMu.Lock()
	jobs = make(map[string]*Job)
	jobsMu.Unlock()

	job1 := NewJob("test_job_1", fiber.Map{})
	job2 := NewJob("test_job_2", fiber.Map{})

	allJobs := GetAllJobs()

	if len(allJobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(allJobs))
	}

	// Verify both jobs are in the list
	foundJob1 := false
	foundJob2 := false
	for _, jobMap := range allJobs {
		if id, ok := jobMap["id"].(string); ok {
			if id == job1.ID {
				foundJob1 = true
			}
			if id == job2.ID {
				foundJob2 = true
			}
		}
	}

	if !foundJob1 || !foundJob2 {
		t.Error("Not all jobs were found in GetAllJobs")
	}
}

func TestCleanupOldJobs(t *testing.T) {
	// Clear existing jobs
	jobsMu.Lock()
	jobs = make(map[string]*Job)
	jobsMu.Unlock()

	// Create an old completed job
	oldJob := NewJob("old_job", fiber.Map{})
	oldJob.Complete(fiber.Map{})
	oldJob.UpdatedAt = time.Now().Add(-25 * time.Hour)

	// Create a recent completed job
	recentJob := NewJob("recent_job", fiber.Map{})
	recentJob.Complete(fiber.Map{})

	// Create a running job (should not be cleaned up)
	runningJob := NewJob("running_job", fiber.Map{})

	CleanupOldJobs()

	// Old completed job should be removed
	if GetJob(oldJob.ID) != nil {
		t.Error("Old completed job should be cleaned up")
	}

	// Recent job should still exist
	if GetJob(recentJob.ID) == nil {
		t.Error("Recent job should not be cleaned up")
	}

	// Running job should still exist
	if GetJob(runningJob.ID) == nil {
		t.Error("Running job should not be cleaned up")
	}
}

func TestJobToMap(t *testing.T) {
	job := NewJob("test_job", fiber.Map{
		"input_key": "input_value",
	})
	job.Complete(fiber.Map{
		"result_key": "result_value",
	})

	jobMap := job.ToMap()

	if jobMap["id"] != job.ID {
		t.Error("ToMap should include job ID")
	}

	if jobMap["type"] != "test_job" {
		t.Error("ToMap should include job type")
	}

	if jobMap["status"] != StatusCompleted {
		t.Error("ToMap should include job status")
	}

	if jobMap["progress"] != 100 {
		t.Error("ToMap should include progress")
	}

	if input, ok := jobMap["input"].(fiber.Map); !ok || input["input_key"] != "input_value" {
		t.Error("ToMap should include input data")
	}

	if result, ok := jobMap["result"].(fiber.Map); !ok || result["result_key"] != "result_value" {
		t.Error("ToMap should include result data")
	}
}

// ---- CancelJob ----

func TestCancelJob_Running(t *testing.T) {
	ClearAllJobs()
	job := NewJob("cancel_test", fiber.Map{})
	job.Start()

	err := CancelJob(job.ID)
	if err != nil {
		t.Errorf("CancelJob() unexpected error: %v", err)
	}
	if job.Status != StatusCancelled {
		t.Errorf("expected status cancelled, got %s", job.Status)
	}
	if job.Error != "Job cancelled by user" {
		t.Errorf("expected cancel error message, got %q", job.Error)
	}
	if job.EndedAt == nil {
		t.Error("EndedAt should be set after cancel")
	}
}

func TestCancelJob_NotFound(t *testing.T) {
	err := CancelJob("non-existent-job-id")
	if err == nil {
		t.Error("CancelJob() expected error for non-existent ID, got nil")
	}
}

func TestCancelJob_AlreadyCompleted(t *testing.T) {
	ClearAllJobs()
	job := NewJob("completed_job", fiber.Map{})
	job.Complete(fiber.Map{"result": "done"})

	// Cancelling a completed job should be a no-op (no error, stays completed)
	err := CancelJob(job.ID)
	if err != nil {
		t.Errorf("CancelJob() on completed job unexpected error: %v", err)
	}
	if job.Status != StatusCompleted {
		t.Errorf("completed job should stay completed, got %s", job.Status)
	}
}

func TestCancelJob_AlreadyFailed(t *testing.T) {
	ClearAllJobs()
	job := NewJob("failed_job", fiber.Map{})
	job.Fail("original error")

	err := CancelJob(job.ID)
	if err != nil {
		t.Errorf("CancelJob() on failed job unexpected error: %v", err)
	}
	if job.Status != StatusFailed {
		t.Errorf("failed job should stay failed, got %s", job.Status)
	}
}

// ---- IsCancelled ----

func TestIsCancelled_NotCancelled(t *testing.T) {
	job := NewJob("running_job", fiber.Map{})
	job.Start()

	if job.IsCancelled() {
		t.Error("IsCancelled() should be false for a running job")
	}
}

func TestIsCancelled_AfterCancel(t *testing.T) {
	job := NewJob("cancel_me", fiber.Map{})
	job.Start()
	job.Cancel()

	if !job.IsCancelled() {
		t.Error("IsCancelled() should be true after Cancel()")
	}
}

func TestIsCancelled_NewJob(t *testing.T) {
	job := NewJob("new_pending", fiber.Map{})
	if job.IsCancelled() {
		t.Error("IsCancelled() should be false for a new pending job")
	}
}

// ---- GetJob not found ----

func TestGetJob_NotFound(t *testing.T) {
	result := GetJob("this-id-does-not-exist")
	if result != nil {
		t.Error("GetJob() should return nil for non-existent ID")
	}
}

// ---- CleanupOldJobs does not remove cancelled jobs ----

func TestCleanupOldJobs_KeesCancelledJobs(t *testing.T) {
	ClearAllJobs()

	cancelledJob := NewJob("cancelled", fiber.Map{})
	cancelledJob.Start()
	cancelledJob.Cancel()
	// Make it look old
	cancelledJob.UpdatedAt = time.Now().Add(-25 * time.Hour)

	CleanupOldJobs()

	// Cancelled jobs should NOT be cleaned up (only completed/failed)
	if GetJob(cancelledJob.ID) == nil {
		t.Error("cancelled jobs should NOT be cleaned up by CleanupOldJobs")
	}
}

// ---- Job uniqueness ----

func TestNewJob_UniqueIDs(t *testing.T) {
	ClearAllJobs()
	j1 := NewJob("type", fiber.Map{})
	j2 := NewJob("type", fiber.Map{})
	if j1.ID == j2.ID {
		t.Error("Two jobs should have different IDs")
	}
}

// ---- CreatedAt/UpdatedAt timing ----

func TestJob_TimestampsSet(t *testing.T) {
	before := time.Now()
	job := NewJob("timing_test", fiber.Map{})
	after := time.Now()

	if job.CreatedAt.Before(before) || job.CreatedAt.After(after) {
		t.Errorf("CreatedAt %v not between %v and %v", job.CreatedAt, before, after)
	}
	if job.UpdatedAt.Before(before) || job.UpdatedAt.After(after) {
		t.Errorf("UpdatedAt %v not between %v and %v", job.UpdatedAt, before, after)
	}
}

// ---- WebSocket / BroadcastJobUpdate tests ----

func TestBroadcastJobUpdate_NilHub(t *testing.T) {
	// Save and reset hub to nil to test early return path
	saved := hub
	hub = nil
	defer func() { hub = saved }()

	job := NewJob("nil_hub_test", fiber.Map{})
	// Must not panic
	BroadcastJobUpdate(job)
}

func TestInitWebSocketHub_SetsHub(t *testing.T) {
	// Save and reset hub
	saved := hub
	hub = nil
	defer func() {
		// Restore (we can't easily stop the goroutine from the old hub)
		if saved != nil {
			hub = saved
		}
	}()

	InitWebSocketHub()
	if hub == nil {
		t.Error("InitWebSocketHub should set hub to non-nil")
	}
}

func TestBroadcastJobUpdate_WithHub(t *testing.T) {
	// Ensure hub is initialized
	if hub == nil {
		InitWebSocketHub()
	}

	job := NewJob("broadcast_test", fiber.Map{"key": "val"})
	job.Complete(fiber.Map{"result": "ok"})

	// Must not panic — message goes to broadcast channel, run() will consume it
	BroadcastJobUpdate(job)

	// Give run() goroutine time to process the broadcast
	time.Sleep(20 * time.Millisecond)
}

func TestBroadcastJobUpdate_ChannelFull(t *testing.T) {
	// Create a temporary hub with an unbuffered broadcast channel to force the default: path
	tempHub := &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan WSMessage), // unbuffered — will block if nobody reads
		register:   make(chan *fiber_websocket.Conn),
		unregister: make(chan *fiber_websocket.Conn),
	}
	saved := hub
	hub = tempHub
	defer func() { hub = saved }()

	job := NewJob("channel_full_test", fiber.Map{})
	// Nobody is reading from tempHub.broadcast — should hit default:
	BroadcastJobUpdate(job)
}
