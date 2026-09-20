//go:build linux

package v4l

import "unsafe"

// The V4L2 structs carry C unions, which cgo -godefs renders as a byte array
// sized to the largest member. Reading or writing a member means
// reinterpreting that array.
//
// Every accessor below converts a pointer rather than copying a value. That
// is not a micro-optimisation: copying the whole union out of a smaller
// member reads past the end of that member, and copying it back in writes
// the reserved tail the kernel expects to find untouched.
//
// These declarations fail to compile if a union member outgrows its union.
var (
	_ [unsafe.Sizeof(Format{}.Fmt) - unsafe.Sizeof(PixFormat{})]struct{}
	_ [unsafe.Sizeof(Format{}.Fmt) - unsafe.Sizeof(PixFormatMplane{})]struct{}
	_ [unsafe.Sizeof(Streamparm{}.Parm) - unsafe.Sizeof(Captureparm{})]struct{}
	_ [unsafe.Sizeof(Streamparm{}.Parm) - unsafe.Sizeof(Outputparm{})]struct{}
	_ [unsafe.Sizeof(Buffer{}.M) - unsafe.Sizeof((*Plane)(nil))]struct{}
	_ [unsafe.Sizeof(Plane{}.M) - unsafe.Sizeof(uint32(0))]struct{}
)

// FmtUnion lists the members of the fmt union in a v4l2_format.
type FmtUnion interface {
	PixFormat | PixFormatMplane
}

// ParmUnion lists the members of the parm union in a v4l2_streamparm.
type ParmUnion interface {
	Captureparm | Outputparm
}

// FrmsizeUnion lists the members of the frame size union in a
// v4l2_frmsizeenum.
type FrmsizeUnion interface {
	FrmsizeDiscrete | FrmsizeStepwise
}

// FrmivalUnion lists the members of the frame interval union in a
// v4l2_frmivalenum.
type FrmivalUnion interface {
	Fract | FrmivalStepwise
}

// As reinterprets the fmt union of f as a T. The result aliases f, so writing
// through it sets the format in place.
func (f *Format) As[T FmtUnion]() *T {
	return (*T)(unsafe.Pointer(&f.Fmt[0]))
}

// As reinterprets the parm union of sp as a T. The result aliases sp, so
// writing through it sets the stream parameters in place.
func (sp *Streamparm) As[T ParmUnion]() *T {
	return (*T)(unsafe.Pointer(&sp.Parm[0]))
}

// As reinterprets the frame size union of e as a T. Consult e.Type before
// choosing T: FRMSIZE_TYPE_DISCRETE means FrmsizeDiscrete, the continuous and
// stepwise types mean FrmsizeStepwise.
func (e *Frmsizeenum) As[T FrmsizeUnion]() *T {
	return (*T)(unsafe.Pointer(&e.Discrete))
}

// As reinterprets the frame interval union of e as a T. Consult e.Type before
// choosing T: FRMIVAL_TYPE_DISCRETE means Fract, the continuous and stepwise
// types mean FrmivalStepwise.
func (e *Frmivalenum) As[T FrmivalUnion]() *T {
	return (*T)(unsafe.Pointer(&e.Discrete))
}

// Planes returns the planes of b.
//
// Multi-planar buffers alias the array installed by SetPlanes. Single-planar
// buffers have no plane array, so one plane is synthesised from the geometry
// the driver reports on the buffer itself.
func (b *Buffer) Planes() []Plane {
	if b.Type == BUF_TYPE_VIDEO_CAPTURE {
		return []Plane{{Bytesused: b.Bytesused, Length: b.Length, M: b.M}}
	}
	return unsafe.Slice(*(**Plane)(unsafe.Pointer(&b.M[0])), b.Length)
}

// SetPlanes points b at planes and records their count in b.Length.
//
// b.M is a byte array, so the pointer stored in it is invisible to the
// garbage collector. The caller must keep planes reachable for as long as the
// driver may write to it, which in practice means holding the slice in a
// field beside the Buffer.
func (b *Buffer) SetPlanes(planes []Plane) {
	*(**Plane)(unsafe.Pointer(&b.M[0])) = unsafe.SliceData(planes)
	b.Length = uint32(len(planes))
}

// MemOffset returns the mmap offset stored in the m union of p.
func (p *Plane) MemOffset() uint32 {
	return *(*uint32)(unsafe.Pointer(&p.M[0]))
}
