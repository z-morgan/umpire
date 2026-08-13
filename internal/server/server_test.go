package server

import (
	"net"
	"testing"
)

// The listener must stay bound to loopback. Umpire serves the repository's
// source, so a wildcard bind would expose it to anyone on the same network.
// This is a one-line address string that a future tidy-up could easily undo.
func TestNewBindsToLoopback(t *testing.T) {
	srv, err := New(0)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.listener.Close()

	tcpAddr, ok := srv.listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address is %T, want *net.TCPAddr", srv.listener.Addr())
	}
	if !tcpAddr.IP.IsLoopback() {
		t.Errorf("listener bound to %s, want a loopback address", tcpAddr.IP)
	}
}
