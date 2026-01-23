package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"videostreamgo/internal/models/instance"
)

// JobType defines the type of background job
type JobType string

const (
	JobTypeProcessVideo        JobType = "process_video"
	JobTypeGenerateThumbnails  JobType = "generate_thumbnails"
	JobTypeTranscodeVideo      JobType = "transcode_video"
	JobTypeExtractMetadata     JobType = "extract_metadata"
	JobTypeCleanupTempFiles    JobType = "cleanup_temp_files"
	JobTypeCleanupFailedUpload JobType = "cleanup_failed_upload"
)

// VideoProcessingPayload contains the payload for video processing jobs
type VideoProcessingPayload struct {
	VideoID    string   `json:"video_id"`
	InstanceID string   `json:"instance_id"`
	Qualities  []string `json:"qualities,omitempty"`
	Priority   int      `json:"priority,omitempty"`
}

// JobWorker handles background job processing
type JobWorker struct {
	processingService VideoProcessingService
	storageService    StorageService
}

// NewJobWorker creates a new job worker
func NewJobWorker(processingService VideoProcessingService, storageService StorageService) *JobWorker {
	return &JobWorker{
		processingService: processingService,
		storageService:    storageService,
	}
}

// ProcessVideoJob handles video processing jobs
func (w *JobWorker) ProcessVideoJob(ctx context.Context, task *Task) error {
	var payload VideoProcessingPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Printf("Processing video: %s", payload.VideoID)

	// Process the video
	if err := w.processingService.ProcessVideo(payload.VideoID); err != nil {
		log.Printf("Failed to process video %s: %v", payload.VideoID, err)
		return err
	}

	log.Printf("Successfully processed video: %s", payload.VideoID)
	return nil
}

// ProcessThumbnailJob handles thumbnail generation jobs
func (w *JobWorker) ProcessThumbnailJob(ctx context.Context, task *Task) error {
	var payload VideoProcessingPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Printf("Generating thumbnails for video: %s", payload.VideoID)

	if err := w.processingService.GenerateThumbnails(payload.VideoID); err != nil {
		log.Printf("Failed to generate thumbnails for video %s: %v", payload.VideoID, err)
		return err
	}

	log.Printf("Successfully generated thumbnails for video: %s", payload.VideoID)
	return nil
}

// ProcessCleanupJob handles cleanup jobs
func (w *JobWorker) ProcessCleanupJob(ctx context.Context, task *Task) error {
	var payload VideoProcessingPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Printf("Cleaning up temp files for video: %s", payload.VideoID)

	if err := w.processingService.CleanupTempFiles(payload.VideoID); err != nil {
		log.Printf("Failed to cleanup temp files for video %s: %v", payload.VideoID, err)
		return err
	}

	// Also cleanup failed uploads
	if err := w.storageService.DeleteVideo(payload.VideoID); err != nil {
		log.Printf("Failed to delete video from storage %s: %v", payload.VideoID, err)
	}

	log.Printf("Successfully cleaned up video: %s", payload.VideoID)
	return nil
}

// Task represents a background task (simplified asynq replacement)
type Task struct {
	Type    string
	Payload []byte
}

// TaskHandler is a function that handles a task
type TaskHandler func(ctx context.Context, task *Task) error

// SimpleJobQueue implements a simple in-memory job queue
type SimpleJobQueue struct {
	tasks    chan *Task
	handlers map[string]TaskHandler
	stopChan chan struct{}
}

// NewSimpleJobQueue creates a simple job queue
func NewSimpleJobQueue(bufferSize int) *SimpleJobQueue {
	return &SimpleJobQueue{
		tasks:    make(chan *Task, bufferSize),
		handlers: make(map[string]TaskHandler),
		stopChan: make(chan struct{}),
	}
}

// RegisterHandler registers a handler for a task type
func (q *SimpleJobQueue) RegisterHandler(taskType string, handler TaskHandler) {
	q.handlers[taskType] = handler
}

// Enqueue adds a task to the queue
func (q *SimpleJobQueue) Enqueue(task *Task) error {
	select {
	case q.tasks <- task:
		return nil
	default:
		return fmt.Errorf("job queue is full")
	}
}

// Start starts the worker goroutines
func (q *SimpleJobQueue) Start(workers int) {
	for i := 0; i < workers; i++ {
		go func() {
			for {
				select {
				case task := <-q.tasks:
					if handler, ok := q.handlers[task.Type]; ok {
						if err := handler(context.Background(), task); err != nil {
							log.Printf("Task %s failed: %v", task.Type, err)
						}
					}
				case <-q.stopChan:
					return
				}
			}
		}()
	}
}

// Stop stops the job queue
func (q *SimpleJobQueue) Stop() {
	close(q.stopChan)
}

// NewVideoProcessingTask creates a new video processing task
func NewVideoProcessingTask(payload *VideoProcessingPayload) (*Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return &Task{
		Type:    string(JobTypeProcessVideo),
		Payload: data,
	}, nil
}

// EnqueueVideoProcessing enqueues a video processing job
func (q *SimpleJobQueue) EnqueueVideoProcessing(videoID, instanceID string, qualities []instance.VideoQuality) error {
	payload := &VideoProcessingPayload{
		VideoID:    videoID,
		InstanceID: instanceID,
		Qualities:  []string{},
		Priority:   5,
	}

	for _, q := range qualities {
		payload.Qualities = append(payload.Qualities, string(q))
	}

	task, err := NewVideoProcessingTask(payload)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return q.Enqueue(task)
}

// EnqueueThumbnailGeneration enqueues a thumbnail generation job
func (q *SimpleJobQueue) EnqueueThumbnailGeneration(videoID, instanceID string) error {
	payload := &VideoProcessingPayload{
		VideoID:    videoID,
		InstanceID: instanceID,
		Priority:   3,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := &Task{
		Type:    string(JobTypeGenerateThumbnails),
		Payload: data,
	}

	return q.Enqueue(task)
}

// EnqueueCleanup enqueues a cleanup job
func (q *SimpleJobQueue) EnqueueCleanup(videoID, instanceID string) error {
	payload := &VideoProcessingPayload{
		VideoID:    videoID,
		InstanceID: instanceID,
		Priority:   1,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := &Task{
		Type:    string(JobTypeCleanupTempFiles),
		Payload: data,
	}

	// Process cleanup after delay
	time.AfterFunc(1*time.Hour, func() {
		q.Enqueue(task)
	})

	return nil
}

// GenerateTaskID generates a unique task ID
func GenerateTaskID() string {
	return uuid.New().String()
}
