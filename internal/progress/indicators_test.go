package progress

import (
	"os"
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	if got := formatDuration(500 * time.Millisecond); got != "500ms" {
		t.Errorf("expected 500ms, got %s", got)
	}
	if got := formatDuration(2 * time.Second); got != "2.0s" {
		t.Errorf("expected 2.0s, got %s", got)
	}
	if got := formatDuration(2 * time.Minute); got != "2.0m" {
		t.Errorf("expected 2.0m, got %s", got)
	}
}

func TestProgressBarLifecycle(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	pb := NewProgressBar(10, "Testing")
	pb.Update(5)
	pb.SetCurrent(7)
	pb.Finish()

	// Restore stdout
	w.Close()
	os.Stdout = old
	_ = r // could read captured output if needed
}

func TestSpinnerStartStop(t *testing.T) {
	s := NewSpinner("Working")
	s.Start()
	time.Sleep(200 * time.Millisecond) // let it tick once
	s.Stop()                           // ensure it shuts down without panic
}
