package jobs

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
	StatusCancelled JobStatus = "cancelled"
)

type Job struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Status     JobStatus  `json:"status"`
	Progress   int        `json:"progress"`          // 0-100
	Message    string     `json:"message,omitempty"` // Current operation message (e.g., "Downloading: Track Name")
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
	Input      fiber.Map  `json:"input"`
	Result     fiber.Map  `json:"result,omitempty"`
	Error      string     `json:"error,omitempty"`
	mu         sync.RWMutex
	cancelChan chan struct{} `json:"-"` // Channel to signal cancellation
}

var (
	jobs   = make(map[string]*Job)
	jobsMu sync.RWMutex
)

func NewJob(jobType string, input fiber.Map) *Job {
	job := &Job{
		ID:         uuid.New().String(),
		Type:       jobType,
		Status:     StatusPending,
		Progress:   0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Input:      input,
		cancelChan: make(chan struct{}),
	}

	jobsMu.Lock()
	jobs[job.ID] = job
	jobsMu.Unlock()

	// Don't auto-start - let caller start after acquiring resources (e.g., semaphore)
	return job
}

func (j *Job) Start() {
	j.mu.Lock()
	defer j.mu.Unlock()

	now := time.Now()
	j.Status = StatusRunning
	j.StartedAt = &now
	j.UpdatedAt = now
}

func (j *Job) UpdateProgress(progress int) {
	j.mu.Lock()
	j.Progress = progress
	j.UpdatedAt = time.Now()
	j.mu.Unlock()

	// Broadcast update via WebSocket
	BroadcastJobUpdate(j)
}

func (j *Job) UpdateMessage(message string) {
	j.mu.Lock()
	j.Message = message
	j.UpdatedAt = time.Now()
	j.mu.Unlock()

	// Broadcast update via WebSocket
	BroadcastJobUpdate(j)
}

func (j *Job) UpdateProgressWithMessage(progress int, message string) {
	j.mu.Lock()
	j.Progress = progress
	j.Message = message
	j.UpdatedAt = time.Now()
	j.mu.Unlock()

	// Broadcast update via WebSocket
	BroadcastJobUpdate(j)
}

func (j *Job) Complete(result fiber.Map) {
	j.mu.Lock()
	now := time.Now()
	j.Status = StatusCompleted
	j.Progress = 100
	j.Result = result
	j.EndedAt = &now
	j.UpdatedAt = now
	j.mu.Unlock()

	// Broadcast completion via WebSocket
	BroadcastJobUpdate(j)
}

func (j *Job) Fail(errorMsg string) {
	j.mu.Lock()
	now := time.Now()
	j.Status = StatusFailed
	j.Error = errorMsg
	j.EndedAt = &now
	j.UpdatedAt = now
	j.mu.Unlock()

	// Broadcast failure via WebSocket
	BroadcastJobUpdate(j)
}

func (j *Job) Cancel() {
	j.mu.Lock()
	if j.Status != StatusRunning {
		j.mu.Unlock()
		return
	}
	now := time.Now()
	j.Status = StatusCancelled
	j.Error = "Job cancelled by user"
	j.EndedAt = &now
	j.UpdatedAt = now
	j.mu.Unlock()

	// Signal cancellation
	close(j.cancelChan)

	// Broadcast cancellation via WebSocket
	BroadcastJobUpdate(j)
}

func (j *Job) IsCancelled() bool {
	select {
	case <-j.cancelChan:
		return true
	default:
		return false
	}
}

func (j *Job) ToMap() fiber.Map {
	j.mu.RLock()
	defer j.mu.RUnlock()

	return fiber.Map{
		"id":         j.ID,
		"type":       j.Type,
		"status":     j.Status,
		"progress":   j.Progress,
		"message":    j.Message,
		"created_at": j.CreatedAt,
		"updated_at": j.UpdatedAt,
		"started_at": j.StartedAt,
		"ended_at":   j.EndedAt,
		"input":      j.Input,
		"result":     j.Result,
		"error":      j.Error,
	}
}

func GetJob(id string) *Job {
	jobsMu.RLock()
	defer jobsMu.RUnlock()
	return jobs[id]
}

func GetAllJobs() []fiber.Map {
	jobsMu.RLock()
	defer jobsMu.RUnlock()

	result := make([]fiber.Map, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, job.ToMap())
	}
	return result
}

func CancelJob(id string) error {
	job := GetJob(id)
	if job == nil {
		return fiber.NewError(404, "Job not found")
	}
	job.Cancel()
	return nil
}

// CleanupOldJobs removes completed/failed jobs older than 24 hours
func CleanupOldJobs() {
	jobsMu.Lock()
	defer jobsMu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for id, job := range jobs {
		if (job.Status == StatusCompleted || job.Status == StatusFailed) && job.UpdatedAt.Before(cutoff) {
			delete(jobs, id)
		}
	}
}

// Start periodic cleanup
func init() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			CleanupOldJobs()
		}
	}()
}

// ClearAllJobs removes all jobs - FOR TESTING ONLY
func ClearAllJobs() {
	jobsMu.Lock()
	defer jobsMu.Unlock()
	jobs = make(map[string]*Job)
}
