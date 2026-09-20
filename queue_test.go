//go:build linux

package v4l

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hnnsgstfssn/v4l/v4l"
)

func TestQueue(t *testing.T) {
	if ok, _ := strconv.ParseBool(os.Getenv("TEST_QUEUE")); !ok {
		t.Skip("skipping queue test; set TEST_QUEUE to run the test")
	}
	cam, err := NewCamera("/dev/video0", 640, 480, v4l.PIX_FMT_MJPEG, 10)
	if err != nil {
		t.Skipf("v4l.NewCamera: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	t.Cleanup(func() {
		if err2 := cam.Close(); err2 != nil {
			t.Error(err2)
		}
	})
	f, err := os.OpenFile("data", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o0644)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	})
	for !isDone(ctx) {
		data, _, err := cam.Frame()
		if err != nil {
			t.Errorf("Camera.Frame: %v", err)
			break
		}
		for i := range data {
			slog.Info("F", "planes", len(data), "index", i, "data", len(data[i]))
			if _, err := f.Write(data[i]); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false
}
