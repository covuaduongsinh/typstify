//go:build !windows

package main

import "syscall"

// exclusiveAddr is a no-op where binding is already exclusive per user
// (no SO_REUSEPORT is set).
func exclusiveAddr(_, _ string, _ syscall.RawConn) error { return nil }
