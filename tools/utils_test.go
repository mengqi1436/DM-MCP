package tools

import "testing"

func TestGetIntAcceptsStringNumbers(t *testing.T) {
	params := map[string]interface{}{
		"port": "5237",
	}

	got := getInt(params, "port", 5236)
	if got != 5237 {
		t.Fatalf("getInt() = %d, want 5237", got)
	}
}
