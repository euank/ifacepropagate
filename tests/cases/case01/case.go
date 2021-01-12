package case01

import (
	"io"
)

type readCloseFrobulator struct {
	io.Reader
}

func (r readCloseFrobulator) Read(b []byte) (int, error) {
	return 0, nil
}

type ptrReadCloseFrobulator struct {
	io.Reader
}

func (r *ptrReadCloseFrobulator) Read(b []byte) (int, error) {
	return 0, nil
}
