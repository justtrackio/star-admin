package internal

import "testing"

func TestClusterNodeMetadata(t *testing.T) {
	handler := &HandlerOverview{}

	node := handler.nodeFromMetadata(map[string]string{
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

func TestClusterNodeMissingMetadata(t *testing.T) {
	handler := &HandlerOverview{}

	node := handler.nodeFromMetadata(map[string]string{"CpuCores": "N/A", "Alive": "unknown"})
	if node.Alive != nil || node.CPUCores != nil || node.Tablets != nil || node.RunningQueries != nil {
		t.Fatalf("unavailable metadata must stay unknown: %+v", node)
	}
}

func TestNodeTypesMetadata(t *testing.T) {
	handler := &HandlerOverview{}
	for _, metadata := range []map[string]string{
		{"Id": "1", "IP": "fe.local", "Role": "LEADER", "Alive": "true", "StartTime": "2026-10-24 10:00:00", "QueryPort": "9030"},
		{"Name": "fe_legacy", "Host": "fe.local", "Role": "FOLLOWER"},
		{"BackendId": "9007199254740993", "IP": "be.local", "TabletNum": "42", "TotalCapacity": "1TB"},
	} {
		node := handler.nodeFromMetadata(metadata)
		if node.ID == "" || node.Host == "" {
			t.Fatalf("missing node identity: %+v", node)
		}

		for key, value := range metadata {
			if node.Details[key] != value {
				t.Fatalf("lost detail %s: %+v", key, node)
			}
		}

		if metadata["StartTime"] != "" && node.LastStartTime != metadata["StartTime"] {
			t.Fatalf("lost frontend start time: %+v", node)
		}

		if node.Role != metadata["Role"] {
			t.Fatalf("lost frontend role: %+v", node)
		}
	}
}

func TestFrontendHeapMetrics(t *testing.T) {
	handler := &HandlerOverview{}
	used, maxHeap, zero, negative := int64(1<<30), int64(4<<30), int64(0), int64(-1)
	nodes := []ClusterNode{
		{Type: "frontend", ID: "1", Details: map[string]string{"Name": "fe-b"}},
		{Type: "frontend", ID: "2", Details: map[string]string{"Name": "fe-a"}},
		{Type: "frontend", Details: map[string]string{"Name": "missing"}},
		{Type: "frontend", Details: map[string]string{"Name": "zero-max"}},
		{Type: "frontend", Details: map[string]string{"Name": "negative"}},
		{Type: "frontend", Details: map[string]string{"Name": "incomplete"}},
		{Type: "backend", Details: map[string]string{"Name": "fe-b"}},
	}
	handler.applyFrontendHeap(nodes, []frontendHeapMetrics{
		{Name: "fe-a", UsedBytes: &zero, MaxBytes: &maxHeap},
		{Name: "fe-b", UsedBytes: &used, MaxBytes: &maxHeap},
		{Name: "zero-max", UsedBytes: &used, MaxBytes: &zero},
		{Name: "negative", UsedBytes: &negative, MaxBytes: &maxHeap},
		{Name: "incomplete", MaxBytes: &maxHeap},
	})

	if nodes[0].MemoryUsed != "25.0 %" || nodes[0].MemoryLimit != "4.00 GiB" || nodes[0].Details["JVM heap used (bytes)"] != "1073741824" {
		t.Fatalf("lost frontend heap measurements: %+v", nodes[0])
	}

	if nodes[1].MemoryUsed != "0.0 %" {
		t.Fatalf("zero usage must stay known: %+v", nodes[1])
	}

	for _, node := range nodes[2:] {
		if node.MemoryUsed != "" {
			t.Fatalf("invalid or unmatched metrics must remain unavailable: %+v", node)
		}
	}
}
