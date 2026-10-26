package internal

import (
	"errors"
	"testing"
)

func TestRoutineLoadSettingsQueryError(t *testing.T) {
	failure := errors.New("configuration permission denied")
	handler := &HandlerTables{client: &routineTasksQueryClient{taskError: failure}}

	response, err := handler.GetRoutineLoadSettings(t.Context())
	if response != nil || !errors.Is(err, failure) {
		t.Fatalf("settings error must remain visible: %v, %v", response, err)
	}
}
