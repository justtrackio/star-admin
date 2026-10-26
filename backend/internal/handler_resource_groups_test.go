package internal

import "testing"

func TestResourceGroupsFromMetadata(t *testing.T) {
	handler := &HandlerResourceGroups{}
	groups := handler.resourceGroupsFromMetadata([]map[string]string{
		{"id": "9007199254740993", "name": "analytics", "cpu_weight": "4", "mem_limit": "50%", "concurrency_limit": "0", "exclusive_cpu_percent": "null", "cpu_weight_percent": "1", "classifiers": "(id=1, user='analytics')", "future_limit": "reported"},
		{"id": "2", "name": "default_wg", "mem_limit": "100%"},
		{"id": "9007199254740993", "name": "analytics", "cpu_weight": "4", "classifiers": "(id=2, db='analytics')"},
	})
	if len(groups) != 2 || groups[0].ID != "9007199254740993" || len(groups[0].Classifiers) != 2 {
		t.Fatalf("must retain IDs and combine all classifiers per group: %+v", groups)
	}
	if groups[0].Properties["exclusive_cpu_percent"] != "" || groups[0].Properties["cpu_weight_percent"] != "1" || groups[0].Properties["future_limit"] != "reported" || groups[0].Properties["concurrency_limit"] != "0" {
		t.Fatalf("must preserve reported limits: %+v", groups[0])
	}
	if _, found := groups[0].Properties["classifiers"]; found {
		t.Fatal("classifiers must be separate from properties")
	}
	if groups[1].Classifiers == nil || len(groups[1].Classifiers) != 0 || groups[1].Properties["cpu_weight"] != "" {
		t.Fatalf("missing metadata must stay unavailable: %+v", groups[1])
	}
	if empty := handler.resourceGroupsFromMetadata(nil); empty == nil || len(empty) != 0 {
		t.Fatalf("empty results must encode as an array: %+v", empty)
	}
}
