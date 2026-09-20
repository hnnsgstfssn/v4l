//go:build linux

package v4l

import (
	"syscall"
	"unsafe"
)

// sys is the SYS_IOCTL entry point. It is a variable so tests can substitute
// a fake without a device node.
var sys syscaller = syscall.Syscall

type syscaller func(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)

// ioctl issues req against fd with arg as the argument pointer, and returns
// the raw syscall.Errno on failure. Callers translate it at their own
// boundary.
func ioctl[T any](fd int, req int, arg *T) error {
	if _, _, errno := sys(syscall.SYS_IOCTL, uintptr(fd), uintptr(req), uintptr(unsafe.Pointer(arg))); errno != 0 {
		return errno
	}
	return nil
}

// EnumFmt implements the VIDIOC_ENUM_FMT ioctl.
func EnumFmt(fd int, e *Fmtdesc) error { return ioctl(fd, VIDIOC_ENUM_FMT, e) }

// EnumFramesizes implements the VIDIOC_ENUM_FRAMESIZES ioctl.
func EnumFramesizes(fd int, e *Frmsizeenum) error { return ioctl(fd, VIDIOC_ENUM_FRAMESIZES, e) }

// EnumFrameintervals implements the VIDIOC_ENUM_FRAMEINTERVALS ioctl.
func EnumFrameintervals(fd int, e *Frmivalenum) error {
	return ioctl(fd, VIDIOC_ENUM_FRAMEINTERVALS, e)
}

// GCtrl implements the VIDIOC_G_CTRL ioctl.
func GCtrl(fd int, c *Control) error { return ioctl(fd, VIDIOC_G_CTRL, c) }

// SCtrl implements the VIDIOC_S_CTRL ioctl.
func SCtrl(fd int, c *Control) error { return ioctl(fd, VIDIOC_S_CTRL, c) }

// GFmt implements the VIDIOC_G_FMT ioctl.
func GFmt(fd int, f *Format) error { return ioctl(fd, VIDIOC_G_FMT, f) }

// SFmt implements the VIDIOC_S_FMT ioctl.
func SFmt(fd int, f *Format) error { return ioctl(fd, VIDIOC_S_FMT, f) }

// GParm implements the VIDIOC_G_PARM ioctl.
func GParm(fd int, sp *Streamparm) error { return ioctl(fd, VIDIOC_G_PARM, sp) }

// SParm implements the VIDIOC_S_PARM ioctl.
func SParm(fd int, sp *Streamparm) error { return ioctl(fd, VIDIOC_S_PARM, sp) }

// Reqbufs implements the VIDIOC_REQBUFS ioctl.
func Reqbufs(fd int, rb *Requestbuffers) error { return ioctl(fd, VIDIOC_REQBUFS, rb) }

// Querycap implements the VIDIOC_QUERYCAP ioctl.
func Querycap(fd int, c *Capability) error { return ioctl(fd, VIDIOC_QUERYCAP, c) }

// Querybuf implements the VIDIOC_QUERYBUF ioctl.
func Querybuf(fd int, b *Buffer) error { return ioctl(fd, VIDIOC_QUERYBUF, b) }

// Qbuf implements the VIDIOC_QBUF ioctl.
func Qbuf(fd int, b *Buffer) error { return ioctl(fd, VIDIOC_QBUF, b) }

// Dqbuf implements the VIDIOC_DQBUF ioctl.
func Dqbuf(fd int, b *Buffer) error { return ioctl(fd, VIDIOC_DQBUF, b) }

// Streamon implements the VIDIOC_STREAMON ioctl.
func Streamon(fd int, typ uint32) error { return ioctl(fd, VIDIOC_STREAMON, &typ) }

// Streamoff implements the VIDIOC_STREAMOFF ioctl.
func Streamoff(fd int, typ uint32) error { return ioctl(fd, VIDIOC_STREAMOFF, &typ) }

// Open opens the V4L2 device node dev for reading and writing.
func Open(dev string) (fd int, _ error) {
	return syscall.Open(dev, syscall.O_RDWR|syscall.O_CLOEXEC, 0)
}

// Close closes fd.
func Close(fd int) error { return syscall.Close(fd) }
