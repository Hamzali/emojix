package emojix

import "testing"

func TestAdd(t *testing.T) {
	result := Add(1, 2) 

	if result != 3 {
		t.Errorf("Expected 3, got %d", result)
	}
}
