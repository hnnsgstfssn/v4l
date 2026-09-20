//go:build linux

package v4l

import (
	"errors"
	"fmt"
	"time"

	v4l2 "github.com/hnnsgstfssn/v4l/v4l"
)

// Encoder is a device that can encode frames.
type Encoder struct {
	output  *Queue
	capture *Queue

	fd int
}

// NewEncoder creates and initializes a new encoding device.
func NewEncoder(dev string, width, height, pixelFormat, frameRate uint32, controls ...v4l2.Control) (*Encoder, error) {
	fd, err := v4l2.Open(dev)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	defer func() {
		if err != nil {
			err = errors.Join(err, v4l2.Close(fd))
		}
	}()

	for _, ctl := range controls {
		if err2 := v4l2.SCtrl(fd, &ctl); err2 != nil {
			return nil, fmt.Errorf("SetCtrl: %w", err2)
		}
	}

	output, err := NewQueue(fd, width, height, pixelFormat, v4l2.BUF_TYPE_VIDEO_OUTPUT_MPLANE, &frameRate)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			err = errors.Join(err, output.Stop())
		}
	}()

	capture, err := NewQueue(fd, width, height, v4l2.PIX_FMT_H264, v4l2.BUF_TYPE_VIDEO_CAPTURE_MPLANE, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			err = errors.Join(err, capture.Stop())
		}
	}()

	if err := output.Start(); err != nil {
		return nil, err
	}

	if err := capture.Start(); err != nil {
		return nil, err
	}

	return &Encoder{
		output:  output,
		capture: capture,
		fd:      fd,
	}, nil
}

// ReadFrame reads an encoded frame from the capture queue (encoder output). It
// blocks until frames are available.
func (enc *Encoder) ReadFrame() (payload [][]byte, t time.Time, _ error) {
	return enc.capture.ReadFrame()
}

// WriteFrame writes an unencoded frame to the output queue (encoder input). It
// blocks until there is a free buffer to write to.
func (enc *Encoder) WriteFrame(payload [][]byte, t time.Time) error {
	return enc.output.WriteFrame(payload, t)
}

// Close will close both capture and output queues and release their resources.
func (enc *Encoder) Close() error {
	if err := errors.Join(
		enc.output.Stop(),
		enc.capture.Stop(),
		v4l2.Close(enc.fd),
	); err != nil {
		return fmt.Errorf("failed to close output or capture queues: %w", err)
	}
	return nil
}
