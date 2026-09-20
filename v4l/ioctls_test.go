//go:build linux

package v4l

import (
	"errors"
	"syscall"
	"testing"
)

// TestIoctls checks that every wrapper reaches the syscall layer with the
// request number it documents, and that a non-zero errno is returned rather
// than swallowed.
func TestIoctls(t *testing.T) {
	tests := []struct {
		name string
		req  int
		call func(fd int) error
	}{
		{"EnumFmt", VIDIOC_ENUM_FMT, func(fd int) error { return EnumFmt(fd, &Fmtdesc{}) }},
		{"EnumFramesizes", VIDIOC_ENUM_FRAMESIZES, func(fd int) error { return EnumFramesizes(fd, &Frmsizeenum{}) }},
		{"EnumFrameintervals", VIDIOC_ENUM_FRAMEINTERVALS, func(fd int) error {
			return EnumFrameintervals(fd, &Frmivalenum{})
		}},
		{"GCtrl", VIDIOC_G_CTRL, func(fd int) error { return GCtrl(fd, &Control{}) }},
		{"SCtrl", VIDIOC_S_CTRL, func(fd int) error { return SCtrl(fd, &Control{}) }},
		{"GFmt", VIDIOC_G_FMT, func(fd int) error { return GFmt(fd, &Format{}) }},
		{"SFmt", VIDIOC_S_FMT, func(fd int) error { return SFmt(fd, &Format{}) }},
		{"GParm", VIDIOC_G_PARM, func(fd int) error { return GParm(fd, &Streamparm{}) }},
		{"SParm", VIDIOC_S_PARM, func(fd int) error { return SParm(fd, &Streamparm{}) }},
		{"Reqbufs", VIDIOC_REQBUFS, func(fd int) error { return Reqbufs(fd, &Requestbuffers{}) }},
		{"Querycap", VIDIOC_QUERYCAP, func(fd int) error { return Querycap(fd, &Capability{}) }},
		{"Querybuf", VIDIOC_QUERYBUF, func(fd int) error { return Querybuf(fd, &Buffer{}) }},
		{"Qbuf", VIDIOC_QBUF, func(fd int) error { return Qbuf(fd, &Buffer{}) }},
		{"Dqbuf", VIDIOC_DQBUF, func(fd int) error { return Dqbuf(fd, &Buffer{}) }},
		{"Streamon", VIDIOC_STREAMON, func(fd int) error { return Streamon(fd, BUF_TYPE_VIDEO_CAPTURE) }},
		{"Streamoff", VIDIOC_STREAMOFF, func(fd int) error { return Streamoff(fd, BUF_TYPE_VIDEO_CAPTURE) }},
	}

	const fd = 7

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotTrap, gotFD, gotReq, gotArg uintptr
			swapSys(t, func(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno) {
				gotTrap, gotFD, gotReq, gotArg = trap, a1, a2, a3
				return 0, 0, 0
			})

			if err := tt.call(fd); err != nil {
				t.Fatalf("%s: unexpected error: %v", tt.name, err)
			}
			if gotTrap != uintptr(syscall.SYS_IOCTL) {
				t.Errorf("trap = %#x, want SYS_IOCTL (%#x)", gotTrap, syscall.SYS_IOCTL)
			}
			if gotFD != fd {
				t.Errorf("fd = %d, want %d", gotFD, fd)
			}
			if gotReq != uintptr(tt.req) {
				t.Errorf("req = %#x, want %#x", gotReq, tt.req)
			}
			if gotArg == 0 {
				t.Error("argp = 0, want a non-nil argument pointer")
			}
		})

		t.Run(tt.name+"/errno", func(t *testing.T) {
			swapSys(t, func(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno) {
				return 0, 0, syscall.EINVAL
			})

			err := tt.call(fd)
			if !errors.Is(err, syscall.EINVAL) {
				t.Errorf("err = %v, want EINVAL", err)
			}
		})
	}
}

// swapSys installs fn as the syscall entry point for the duration of t.
func swapSys(t *testing.T, fn syscaller) {
	t.Helper()
	prev := sys
	sys = fn
	t.Cleanup(func() { sys = prev })
}
