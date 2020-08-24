package example

import (
	"io"
	"net"
	"syscall"
)

type ifacepropagateIfaceAlias0 interface {
	syscall.Conn
}

func (l *closeLoggedConn) propogateInterfaces() net.Conn {
	_, ok0 := l.Conn.(io.ReaderFrom)
	_, ok1 := l.Conn.(ifacepropagateIfaceAlias0)
	switch {
	case ok0 && ok1:
		return struct {
			net.Conn
			io.ReaderFrom
			ifacepropagateIfaceAlias0
		}{l, l, l}
	case !ok0 && ok1:
		return struct {
			net.Conn
			ifacepropagateIfaceAlias0
		}{l, l}
	case ok0 && !ok1:
		return struct {
			net.Conn
			io.ReaderFrom
		}{l, l}
	case !ok0 && !ok1:
		return struct {
			net.Conn
		}{l}
	default:
		panic("unreachable")
	}
}
func (l *closeLoggedConn) ReadFrom(r io.Reader) (n int64, err error) {
	return l.Conn.(io.ReaderFrom).ReadFrom(r)
}
func (l *closeLoggedConn) SyscallConn() ( syscall.RawConn,  error) {
	return l.Conn.(ifacepropagateIfaceAlias0).SyscallConn()
}
