package server

import (
	"net"
	"testing"
	"time"
)

func TestDPServerDial(t *testing.T) {
	s := NewDPServer("", "", "", "")

	l, err := net.Listen("unix", socketName)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(2 * time.Second)
		if err := s.srv.Serve(l); err != nil {
			t.Error(err)
			return
		}
	}()

	_, err = s.dial(socketName, time.Second*5)
	if err != nil {
		t.Fatal(err)
	}
}
