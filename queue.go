//go:build linux

package v4l

import (
	"errors"
	"fmt"
	"syscall"
	"time"

	v4l2 "github.com/hnnsgstfssn/v4l/v4l"
)

// Queue is a model of a v4l input or output queue.
type Queue struct {
	Now func() time.Time

	buffers Buffers

	// fd is the file descriptor that the queue belongs to.
	fd int

	next int

	// typ is the buffer type of allocated buffers.
	typ uint32
}

// Stop will close the queue stream and free up allocated buffers.
func (q *Queue) Stop() error {
	if err := errors.Join(
		v4l2.Streamoff(q.fd, q.typ),
		FreeBuffers(q.fd, q.typ, q.buffers),
	); err != nil {
		return fmt.Errorf("failed to cleanup: %w", err)
	}

	return nil
}

// Start begins streaming on the queue.
func (q *Queue) Start() error {
	return v4l2.Streamon(q.fd, q.typ)
}

// ReadFrame dequeues a frame and returns the payload indexed by buffer and
// plane, along with the capture time as reported by the V4L subsystem.
func (q *Queue) ReadFrame() (payload [][]byte, t time.Time, _ error) {
	b := q.buffers[q.next]

	if err := v4l2.Dqbuf(q.fd, b.B); err != nil {
		return nil, time.Time{}, err
	}

	t = q.Now()

	planes := b.B.Planes()
	payload = make([][]byte, len(planes))

	for i, plane := range planes {
		payload[i] = make([]byte, plane.Bytesused-plane.Offset)

		copy(payload[i], b.Payload[i][plane.Offset:plane.Bytesused])
	}

	if err := v4l2.Qbuf(q.fd, b.B); err != nil {
		return nil, time.Time{}, err
	}

	q.next = (q.next + 1) % len(q.buffers)

	return payload, t, nil
}

// WriteFrame enqueues a frame for encoding. The payload should be indexed by
// plane and the timestamp will be copied to the encoded output frame. Note
// that the bytes will be copied from payload into device memory.
func (q *Queue) WriteFrame(payload [][]byte, t time.Time) error {
	b := q.buffers[q.next]

	if err := v4l2.Dqbuf(q.fd, b.B); err != nil {
		return err
	}

	b.B.Flags |= v4l2.BUF_FLAG_TIMESTAMP_COPY
	b.B.Timestamp = syscall.NsecToTimeval(t.UnixNano())

	for i := range payload {
		copy(b.Payload[i], payload[i])
	}

	if err := v4l2.Qbuf(q.fd, b.B); err != nil {
		return err
	}

	q.next = (q.next + 1) % len(q.buffers)

	return nil
}

// NewQueue creates a new queue.
func NewQueue(fd int, width, height, pixelFormat, typ uint32, frameRate *uint32) (*Queue, error) {
	const bufCount uint32 = 5

	f := v4l2.Format{Type: typ}

	if err := v4l2.GFmt(fd, &f); err != nil {
		return nil, err
	}

	// The driver may clamp or substitute what it is asked for, so the format
	// is read back after setting it and the plane count taken from the result
	// rather than from the request.
	switch typ {
	case v4l2.BUF_TYPE_VIDEO_OUTPUT_MPLANE, v4l2.BUF_TYPE_VIDEO_CAPTURE_MPLANE:
		pf := f.As[v4l2.PixFormatMplane]()
		pf.Width, pf.Height, pf.Pixelformat = width, height, pixelFormat
	case v4l2.BUF_TYPE_VIDEO_CAPTURE:
		pf := f.As[v4l2.PixFormat]()
		pf.Width, pf.Height, pf.Pixelformat = width, height, pixelFormat
	}

	if err := v4l2.SFmt(fd, &f); err != nil {
		return nil, err
	}

	if err := v4l2.GFmt(fd, &f); err != nil {
		return nil, err
	}

	planeCount := uint32(1)
	switch typ {
	case v4l2.BUF_TYPE_VIDEO_OUTPUT_MPLANE, v4l2.BUF_TYPE_VIDEO_CAPTURE_MPLANE:
		planeCount = uint32(f.As[v4l2.PixFormatMplane]().Num_planes)
	}

	if frameRate != nil {
		sp := v4l2.Streamparm{Type: typ}
		sp.As[v4l2.Outputparm]().Timeperframe = v4l2.Fract{Numerator: 1, Denominator: *frameRate}

		if err := v4l2.SParm(fd, &sp); err != nil {
			return nil, fmt.Errorf("could not set framerate: %w", err)
		}
	}

	buffers, err := NewBuffers(fd, bufCount, planeCount, typ)
	if err != nil {
		return nil, err
	}

	if err := enqueueBuffers(fd, buffers); err != nil {
		return nil, errors.Join(err, FreeBuffers(fd, typ, buffers))
	}

	return &Queue{
		Now:     time.Now,
		buffers: buffers,
		fd:      fd,
		typ:     typ,
	}, nil
}
