package main

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// soExclusiveAddrUse is Winsock's SO_EXCLUSIVEADDRUSE, ((int)(~SO_REUSEADDR)).
const soExclusiveAddrUse = ^4

// exclusiveAddr makes the listening socket exclusive (SO_EXCLUSIVEADDRUSE):
// no other process can bind the same port alongside it and receive its
// connections.
func exclusiveAddr(_, _ string, c syscall.RawConn) error {
	var serr error
	err := c.Control(func(fd uintptr) {
		serr = windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, soExclusiveAddrUse, 1)
	})
	if err != nil {
		return err
	}
	return serr
}
