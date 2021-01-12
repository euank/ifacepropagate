package case01

import (
	"io"
)

type readFrobulator struct {
	io.Reader
}

func (r readFrobulator) Read(b []byte) (int, error) {
	return 0, nil
}

type ptrReadFrobulator struct {
	io.Reader
}

func (r *ptrReadFrobulator) Read(b []byte) (int, error) {
	return 0, nil
}
