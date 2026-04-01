package job

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/google/uuid"
	"github.com/jollaman999/utils/fileutil"
)

type Manager struct {
	mutex      sync.RWMutex
	jobs       map[string]*Info
	workerPool chan func()
	logRoot    string
}

var DefaultManager *Manager

// AddJobLogSafe appends a line to the job log file when DefaultManager is initialized.
// It is a no-op if the job manager was not started (e.g. K8s migration disabled).
func AddJobLogSafe(jobID, message string) {
	if DefaultManager == nil {
		return
	}
	_ = DefaultManager.AddJobLog(jobID, message)
}

func InitDefaultManager(workerCount int, logRoot string) error {
	manager, err := NewManager(workerCount, logRoot)
	if err != nil {
		return err
	}

	DefaultManager = manager
	return nil
}

func NewManager(workerCount int, logRoot string) (*Manager, error) {
	if workerCount < 1 {
		workerCount = 1
	}

	err := fileutil.CreateDirIfNotExist(logRoot)
	if err != nil {
		return nil, err
	}

	manager := &Manager{
		jobs:       make(map[string]*Info),
		workerPool: make(chan func(), workerCount*2),
		logRoot:    logRoot,
	}

	for i := 0; i < workerCount; i++ {
		go func() {
			for task := range manager.workerPool {
				if task != nil {
					task()
				}
			}
		}()
	}

	return manager, nil
}

func NewJobID(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.NewString())
}

func (m *Manager) CreateJob(jobType, resourceType, resourceName string, metadata map[string]interface{}) (*Info, error) {
	jobID := NewJobID(jobType)
	now := time.Now()
	logPath := m.LogFilePath(jobID)

	info := &Info{
		JobID:        jobID,
		JobType:      jobType,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Status:       StatusPending,
		Progress:     0,
		Message:      "Job created",
		Metadata:     metadata,
		LogPath:      logPath,
		StartedAt:    now,
		UpdatedAt:    now,
	}

	m.mutex.Lock()
	m.jobs[jobID] = info
	m.mutex.Unlock()

	job := &model.JobExecution{
		JobID:        jobID,
		JobType:      jobType,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Status:       string(StatusPending),
		Progress:     0,
		Message:      info.Message,
		Metadata:     BuildMetadataString(metadata),
		LogPath:      logPath,
		StartedAt:    now,
		UpdatedAt:    now,
	}

	_, err := dao.CreateExecution(job)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func (m *Manager) LogFilePath(jobID string) string {
	return filepath.Join(m.logRoot, jobID+".log")
}

func (m *Manager) UpdateJobStatus(jobID string, status Status, progress int, message string) error {
	m.mutex.Lock()
	info, exists := m.jobs[jobID]
	if exists {
		info.Status = status
		info.Progress = progress
		info.Message = message
		info.UpdatedAt = time.Now()
		if status == StatusCompleted || status == StatusFailed {
			info.FinishedAt = info.UpdatedAt
		}
	}
	m.mutex.Unlock()

	job, err := dao.GetExecution(jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job execution not found: %s", jobID)
	}

	job.Status = string(status)
	job.Progress = progress
	job.Message = message
	job.UpdatedAt = time.Now()
	if status == StatusCompleted || status == StatusFailed {
		job.FinishedAt = job.UpdatedAt
	}

	return dao.UpdateExecution(job)
}

func (m *Manager) AddJobLog(jobID string, message string) error {
	job, err := dao.GetExecution(jobID)
	if err != nil {
		return err
	}

	err = fileutil.CreateDirIfNotExist(filepath.Dir(job.LogPath))
	if err != nil {
		return err
	}

	fp, err := os.OpenFile(job.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer func() {
		_ = fp.Close()
	}()

	_, err = fmt.Fprintf(fp, "%s %s\n", time.Now().Format(time.RFC3339), message)
	return err
}

func (m *Manager) FailJob(jobID string, err error) error {
	updateErr := m.UpdateJobStatus(jobID, StatusFailed, 0, err.Error())

	job, getErr := dao.GetExecution(jobID)
	if getErr == nil {
		job.ErrorMessage = err.Error()
		job.UpdatedAt = time.Now()
		job.FinishedAt = job.UpdatedAt
		_ = dao.UpdateExecution(job)
	}

	_ = m.AddJobLog(jobID, "Job failed: "+err.Error())
	return updateErr
}

func (m *Manager) CompleteJob(jobID string, message string) error {
	_ = m.AddJobLog(jobID, message)
	return m.UpdateJobStatus(jobID, StatusCompleted, 100, message)
}

func (m *Manager) Submit(task func()) {
	select {
	case m.workerPool <- task:
	default:
		go task()
	}
}

func (m *Manager) GetJob(jobID string) (*Info, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	info, exists := m.jobs[jobID]
	return info, exists
}
