//go:build linux

package v4l

import (
	"testing"
	"unsafe"
)

// TestFormatAs checks that writing a union member through As touches only
// that member. The kernel reads the reserved tail of the fmt union, so
// clobbering it with whatever followed the source value is not harmless.
func TestFormatAs(t *testing.T) {
	tests := []struct {
		name  string
		write func(*Format)
		size  uintptr
	}{
		{
			name:  "PixFormat",
			size:  unsafe.Sizeof(PixFormat{}),
			write: func(f *Format) { f.As[PixFormat]().Width = 640 },
		},
		{
			name:  "PixFormatMplane",
			size:  unsafe.Sizeof(PixFormatMplane{}),
			write: func(f *Format) { f.As[PixFormatMplane]().Width = 640 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Format{}
			for i := range f.Fmt {
				f.Fmt[i] = 0xAB
			}

			tt.write(&f)

			for i := tt.size; i < uintptr(len(f.Fmt)); i++ {
				if f.Fmt[i] != 0xAB {
					t.Fatalf("byte %d past the %s member = %#x, want 0xAB: the write ran past the member", i, tt.name, f.Fmt[i])
				}
			}
		})
	}
}

func TestFormatAsRoundTrip(t *testing.T) {
	f := Format{Type: BUF_TYPE_VIDEO_CAPTURE}

	pf := f.As[PixFormat]()
	pf.Width, pf.Height, pf.Pixelformat = 1280, 720, PIX_FMT_H264

	got := f.As[PixFormat]()
	if got.Width != 1280 || got.Height != 720 || got.Pixelformat != PIX_FMT_H264 {
		t.Errorf("got %dx%d fmt %#x, want 1280x720 fmt %#x", got.Width, got.Height, got.Pixelformat, PIX_FMT_H264)
	}
	if f.Type != BUF_TYPE_VIDEO_CAPTURE {
		t.Errorf("Type = %d, want it untouched", f.Type)
	}
}

func TestStreamparmAs(t *testing.T) {
	sp := Streamparm{Type: BUF_TYPE_VIDEO_CAPTURE}
	for i := range sp.Parm {
		sp.Parm[i] = 0xCD
	}

	sp.As[Outputparm]().Timeperframe = Fract{Numerator: 1, Denominator: 30}

	if got := sp.As[Outputparm]().Timeperframe; got.Numerator != 1 || got.Denominator != 30 {
		t.Errorf("Timeperframe = %+v, want {1 30}", got)
	}
	for i := unsafe.Sizeof(Outputparm{}); i < uintptr(len(sp.Parm)); i++ {
		if sp.Parm[i] != 0xCD {
			t.Fatalf("byte %d past Outputparm = %#x, want 0xCD", i, sp.Parm[i])
		}
	}
}

func TestBufferPlanes(t *testing.T) {
	t.Run("multiplanar", func(t *testing.T) {
		b := Buffer{Type: BUF_TYPE_VIDEO_CAPTURE_MPLANE}
		planes := make([]Plane, 2)
		planes[0].Bytesused = 111
		planes[1].Bytesused = 222

		b.SetPlanes(planes)

		if b.Length != 2 {
			t.Errorf("Length = %d, want 2", b.Length)
		}

		got := b.Planes()
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if cap(got) < len(got) {
			t.Errorf("cap = %d, len = %d: the slice header is malformed", cap(got), len(got))
		}
		if got[0].Bytesused != 111 || got[1].Bytesused != 222 {
			t.Errorf("Bytesused = %d, %d; want 111, 222", got[0].Bytesused, got[1].Bytesused)
		}

		// The driver writes into the installed array, so Planes must alias it
		// rather than hand back a copy.
		got[1].Bytesused = 333
		if planes[1].Bytesused != 333 {
			t.Error("Planes returned a copy; the driver's writes would be invisible")
		}
	})

	t.Run("singleplanar", func(t *testing.T) {
		b := Buffer{Type: BUF_TYPE_VIDEO_CAPTURE, Bytesused: 4096, Length: 8192}
		b.M = [8]byte{0x00, 0x10, 0x00, 0x00}

		got := b.Planes()
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].Bytesused != 4096 || got[0].Length != 8192 {
			t.Errorf("plane = {%d, %d}, want {4096, 8192}", got[0].Bytesused, got[0].Length)
		}
		if want := uint32(0x1000); got[0].MemOffset() != want {
			t.Errorf("MemOffset = %#x, want %#x", got[0].MemOffset(), want)
		}
	})

	t.Run("empty", func(t *testing.T) {
		b := Buffer{Type: BUF_TYPE_VIDEO_CAPTURE_MPLANE}
		b.SetPlanes(nil)
		if got := b.Planes(); len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})
}

func TestFrmsizeAs(t *testing.T) {
	e := Frmsizeenum{Type: FRMSIZE_TYPE_DISCRETE}
	e.As[FrmsizeDiscrete]().Width = 640

	if got := e.As[FrmsizeDiscrete]().Width; got != 640 {
		t.Errorf("Width = %d, want 640", got)
	}
	if got := e.As[FrmsizeStepwise]().Min_width; got != 640 {
		t.Errorf("Min_width = %d, want 640: both members share the union head", got)
	}
}

func TestFrmivalAs(t *testing.T) {
	e := Frmivalenum{Type: FRMIVAL_TYPE_STEPWISE}
	e.As[FrmivalStepwise]().Step = Fract{Numerator: 1, Denominator: 15}

	if got := e.As[FrmivalStepwise]().Step; got.Denominator != 15 {
		t.Errorf("Step = %+v, want denominator 15", got)
	}
}
