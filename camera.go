//go:build linux

package v4l

import (
	"errors"
	"fmt"
	"time"

	v4l2 "github.com/hnnsgstfssn/v4l/v4l"
)

// Camera is a representation of a V4L camera device.
type Camera struct {
	// capture is the queue used for capturing.
	capture *Queue

	// fd is the file descriptor of the opened device.
	fd int
}

// NewCamera creates and initializes a new capture device.
func NewCamera(dev string, width, height, pixelFormat, frameRate uint32, controls ...v4l2.Control) (*Camera, error) {
	fd, err := v4l2.Open(dev)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			err = errors.Join(err, v4l2.Close(fd))
		}
	}()

	for _, ctl := range controls {
		if err2 := v4l2.SCtrl(fd, &ctl); err2 != nil {
			return nil, err2
		}
	}

	capture, err := NewQueue(fd, width, height, pixelFormat, v4l2.BUF_TYPE_VIDEO_CAPTURE, &frameRate)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			err = errors.Join(err, capture.Stop())
		}
	}()

	if err := capture.Start(); err != nil {
		return nil, err
	}

	return &Camera{
		capture: capture,
		fd:      fd,
	}, nil
}

// Close will stop the capturing and release allocated resources.
func (c *Camera) Close() error {
	if err := c.capture.Stop(); err != nil {
		return fmt.Errorf("queue close: %w", err)
	}
	if err := v4l2.Close(c.fd); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	return nil
}

// Frame dequeues a frame from the capture stream and returns the payload
// indexed by buffer and plane, along with the capture time as reported by the
// V4L subsystem.
func (c *Camera) Frame() (payload [][]byte, t time.Time, _ error) {
	return c.capture.ReadFrame()
}
