package internal

type RoutineLoadTasksInput struct {
	Database string `uri:"database" binding:"required"`
	ID       string `uri:"id" binding:"required,numeric"`
}

type RoutineLoadTask struct {
	ID                string `json:"id"`
	JobID             string `json:"jobId"`
	TransactionID     string `json:"transactionId"`
	TransactionStatus string `json:"transactionStatus"`
	Created           string `json:"created"`
	LastScheduled     string `json:"lastScheduled"`
	ExecuteStarted    string `json:"executeStarted"`
	Timeout           string `json:"timeout"`
	NodeID            string `json:"nodeId"`
	Progress          string `json:"progress"`
	Message           string `json:"message"`
	Warehouse         string `json:"warehouse"`
}

type RoutineLoadTasksResponse struct {
	Tasks []RoutineLoadTask `json:"tasks"`
}
