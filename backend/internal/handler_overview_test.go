package internal

import "testing"

func TestComputeNodeMetadata(t *testing.T) {
	handler := &HandlerOverview{}

	node := handler.computeNodeFromMetadata(map[string]string{
		"ComputeNodeId": "9007199254740993", "IP": "cn-0.cluster.local", "Alive": "false",
		"ClusterDecommissioned": "true", "CpuCores": "8", "NumRunningQueries": "0",
		"MemUsedPct": "29.67 %", "MemLimit": "48.600GB", "ErrMsg": "heartbeat failed",
		"FutureColumn": "ignored",
	})
	if node.ID != "9007199254740993" || node.Alive == nil || *node.Alive || !node.Decommissioned {
		t.Fatalf("lost node identity or status: %+v", node)
	}

	if node.CPUCores == nil || *node.CPUCores != 8 || node.RunningQueries == nil || *node.RunningQueries != 0 {
		t.Fatalf("lost numeric metadata: %+v", node)
	}

	if node.MemoryUsed != "29.67 %" || node.MemoryLimit != "48.600GB" || node.Error != "heartbeat failed" {
		t.Fatalf("lost reported metadata: %+v", node)
	}
}

func TestComputeNodeMissingMetadata(t *testing.T) {
	handler := &HandlerOverview{}

	node := handler.computeNodeFromMetadata(map[string]string{"CpuCores": "N/A", "Alive": "unknown"})
	if node.Alive != nil || node.CPUCores != nil || node.Tablets != nil || node.RunningQueries != nil {
		t.Fatalf("unavailable metadata must stay unknown: %+v", node)
	}
}
