package internal

import "encoding/json"

// MaterializedViewTasksInput identifies the view whose retained task runs are listed.
type MaterializedViewTasksInput struct {
	Database string `uri:"database" binding:"required"`
	ID       string `uri:"id" binding:"required,numeric"`
}

// MaterializedViewTasksResponse contains the latest retained refresh task runs.
type MaterializedViewTasksResponse struct {
	Tasks []MaterializedViewTask `json:"tasks"`
}

type materializedViewTaskExtra struct {
	Partitions       []string `json:"mvPartitionsToRefresh"`
	ProcessStartTime int64    `json:"processStartTime"`
}

// MaterializedViewTask describes a single refresh task run.
type MaterializedViewTask struct {
	Partitions     []string `json:"partitions" db:"-"`
	QueueTime      *float64 `json:"queueTime" db:"-"`
	ProcessingTime *float64 `json:"processingTime" db:"-"`
	ExtraMessage   string   `json:"-" db:"extra_message"`
	CreatedMillis  *int64   `json:"-" db:"created_millis"`
	FinishedMillis *int64   `json:"-" db:"finished_millis"`
	ObservedMillis int64    `json:"-" db:"observed_millis"`
	QueryID        string   `json:"queryId" db:"query_id"`
	State          string   `json:"state" db:"state"`
	Created        string   `json:"created" db:"created"`
	Finished       string   `json:"finished" db:"finished"`
	Progress       string   `json:"progress" db:"progress"`
	Error          string   `json:"error" db:"error_message"`
}

func (task *MaterializedViewTask) enrich() {
	var extra materializedViewTaskExtra
	if err := json.Unmarshal([]byte(task.ExtraMessage), &extra); err == nil {
		task.Partitions = extra.Partitions
	} else {
		extra = materializedViewTaskExtra{}
	}

	start := extra.ProcessStartTime
	if start > 0 {
		task.QueueTime = task.duration(task.CreatedMillis, start)
		if task.FinishedMillis != nil {
			task.ProcessingTime = task.duration(&start, *task.FinishedMillis)
		} else if task.State == "RUNNING" {
			task.ProcessingTime = task.duration(&start, task.ObservedMillis)
		}
	} else if task.State == "PENDING" {
		task.QueueTime = task.duration(task.CreatedMillis, task.ObservedMillis)
	}
}

func (task *MaterializedViewTask) duration(start *int64, end int64) *float64 {
	if start == nil || *start <= 0 || end < *start {
		return nil
	}

	seconds := float64(end-*start) / 1000

	return &seconds
}
