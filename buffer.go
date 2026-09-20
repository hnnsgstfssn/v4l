//go:build linux

package v4l

import (
	"errors"
	"fmt"
	"syscall"

	v4l2 "github.com/hnnsgstfssn/v4l/v4l"
)

type mappedBuffer struct {
	B *v4l2.Buffer

	// Planes backs the plane pointer stored in B.M. That field is a byte
	// array, so the pointer in it is invisible to the garbage collector:
	// this field is what keeps the array alive while the driver writes to
	// it. Nil for single-planar buffers, which have no plane array.
	Planes []v4l2.Plane

	Payload [][]byte
}

type Buffers []*mappedBuffer

func requestBuffers(fd int, count, typ uint32) error {
	rb := v4l2.Requestbuffers{
		Count:  count,
		Type:   typ,
		Memory: v4l2.MEMORY_MMAP,
	}

	if err := v4l2.Reqbufs(fd, &rb); err != nil {
		if errors.Is(err, syscall.EINVAL) {
			return fmt.Errorf("MEMORY_MMAP unsupported")
		}
		return err
	}
	if rb.Count == 0 {
		return fmt.Errorf("driver out of memory")
	}
	return nil
}

func NewBuffers(fd int, count, planes, typ uint32) (Buffers, error) {
	if err := requestBuffers(fd, count, typ); err != nil {
		return nil, err
	}

	bufs := make(Buffers, 0, count)

	for i := range count {
		buf := &mappedBuffer{
			B: &v4l2.Buffer{
				Type:   typ,
				Memory: v4l2.MEMORY_MMAP,
				Index:  i,
			},
			Payload: make([][]byte, planes),
		}

		switch typ {
		case v4l2.BUF_TYPE_VIDEO_CAPTURE_MPLANE, v4l2.BUF_TYPE_VIDEO_OUTPUT_MPLANE:
			buf.Planes = make([]v4l2.Plane, planes)
			buf.B.SetPlanes(buf.Planes)
		}

		bufs = append(bufs, buf)
	}

	if err := queryBuffers(fd, bufs); err != nil {
		return nil, errors.Join(err, FreeBuffers(fd, typ, bufs))
	}

	// A partial mapping still owns whatever it managed to map, so unwind
	// through the same path that a caller would use.
	if err := mapBuffers(fd, bufs); err != nil {
		return nil, errors.Join(err, FreeBuffers(fd, typ, bufs))
	}
	return bufs, nil
}

func enqueueBuffers(fd int, buffers Buffers) error {
	for _, b := range buffers {
		if err := v4l2.Qbuf(fd, b.B); err != nil {
			return err
		}
	}
	return nil
}

func queryBuffers(fd int, buffers Buffers) error {
	for _, b := range buffers {
		if err := v4l2.Querybuf(fd, b.B); err != nil {
			return err
		}
	}
	return nil
}

// mapBuffers maps every plane of every buffer into the process address space.
func mapBuffers(fd int, buffers Buffers) error {
	const prot = syscall.PROT_READ | syscall.PROT_WRITE

	for _, buf := range buffers {
		for j, plane := range buf.B.Planes() {
			mapped, err := syscall.Mmap(fd, int64(plane.MemOffset()), int(plane.Length), prot, syscall.MAP_SHARED)
			if err != nil {
				return fmt.Errorf("mmap: %w", err)
			}

			buf.Payload[j] = mapped
		}
	}

	return nil
}

// FreeBuffers deallocates previously allocated payload buffers.
func FreeBuffers(fd int, typ uint32, buffers Buffers) error {
	var errs []error
	for _, buf := range buffers {
		for j, plane := range buf.Payload {
			if plane == nil {
				continue
			}
			if err := syscall.Munmap(plane); err != nil {
				errs = append(errs, err)
			}
			buf.Payload[j] = nil
		}
		buf.Payload = nil
	}

	rb := v4l2.Requestbuffers{
		Count:  0,
		Type:   typ,
		Memory: v4l2.MEMORY_MMAP,
	}
	return errors.Join(append(errs, v4l2.Reqbufs(fd, &rb))...)
}
