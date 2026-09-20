//go:build linux

package v4l

//go:generate sh -c "go tool cgo -godefs types.go > v4l_linux_$GOARCH.go && rm -r _obj"
