package ptz

import (
	"net"
	"testing"
	"time"
)

func TestDeadlines(t *testing.T) {
	_, err := Connect("198.18.0.254:5678", TimeoutOption(time.Millisecond))
	if err == nil {
		t.Errorf("Wanted a network timeout")
	} else {
		if err, ok := err.(net.Error); ok {
			if !err.Timeout() {
				t.Errorf("Wanted a network timeout")
			}
		} else {
			t.Errorf("Wanted timeout error got %T:%v", err, err)
		}
	}

	l, _ := net.Listen("tcp", "localhost:12345")
	go func() {
		l.Accept()
		l.Close()
	}()

	_, err = Connect("localhost:12345")
	if err != nil {
		t.Errorf("Wanted no error, got %v", err)
	}
}
