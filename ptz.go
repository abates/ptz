package ptz

import (
	"net"
	"time"

	"github.com/abates/ptz/visca"
)

var (
	InqZoomPos    visca.Command = visca.Command{0x04, 0x47}
	InqPanTiltPos               = visca.Command{0x06, 0x12}
)

type Info struct {
	PanPos  int
	TiltPos int
	ZoomPos int
}

type Camera interface {
	Query() (Info, error)
}

type Option func(*camera)

func TimeoutOption(timeout time.Duration) Option {
	return func(cam *camera) {
		cam.timeout = timeout
	}
}

type camera struct {
	addr    string
	timeout time.Duration
	conn    visca.Conn
}

func Connect(addr string, options ...Option) (Camera, error) {
	cam := &camera{
		addr: addr,
	}

	for _, option := range options {
		option(cam)
	}

	return cam, cam.connect()
}

func (cam *camera) connect() error {
	conn, err := dial("tcp", cam.addr, cam.timeout)
	if err == nil {
		cam.conn = visca.New(conn, conn)
	}
	return err
}

func dial(network, address string, timeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout(network, address, timeout)
	if err == nil {
		conn = &connectionWithDeadline{conn, timeout}
	}

	return conn, err
}

type connectionWithDeadline struct {
	net.Conn
	timeout time.Duration
}

func (conn *connectionWithDeadline) Read(b []byte) (n int, err error) {
	err = conn.SetReadDeadline(time.Now().Add(conn.timeout))
	if err == nil {
		n, err = conn.Conn.Read(b)
	}
	return
}

func (cam *camera) Query() (info Info, err error) {
	pkt, err := cam.conn.SendInquiry(InqZoomPos)
	if err == nil {
		info.ZoomPos = pkt.Value(0, 4)

		pkt, err = cam.conn.SendInquiry(InqPanTiltPos)
		if err == nil {
			info.PanPos = pkt.Value(0, 4)
			info.TiltPos = pkt.Value(4, 4)
		}
	}
	return
}
