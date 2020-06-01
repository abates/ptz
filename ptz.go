package ptz

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"time"
)

const (
	//ViscaPort int = 1259
	ViscaPort int = 5678
)

const (
	HeaderByte     byte = 0x81
	CommandByte    byte = 0x01
	InquiryByte    byte = 0x09
	TerminatorByte byte = 0xff
)

type Header byte

type Type int

const (
	Cmd Type = 0x02
	Inq Type = 0x09
)

const (
	CmdCompletion byte = 0x50
	CmdAck        byte = 0x40
	CmdErr        byte = 0x60

	SyntaxError          byte = 0x02
	CommandBufferFull         = 0x03
	CommandCanceled           = 0x04
	NoSocket                  = 0x05
	CommandNotExecutable      = 0x06
)

var (
	ErrSyntax        = errors.New("Syntax Error")
	ErrBufferFull    = errors.New("Command Buffer Full")
	ErrCanceled      = errors.New("Command canceled")
	ErrNoSocket      = errors.New("Cancel failed (no socket)")
	ErrNotExecutable = errors.New("Cannot execute command")
	ErrUnknown       = errors.New("Unknown error occurred")
)

type Command [2]byte

var (
	InqZoomPos    Command = Command{0x04, 0x47}
	InqPanTiltPos Command = Command{0x06, 0x12}
)

type Packet []byte

func (p Packet) Header() Header   { return Header(p[0]) }
func (p Packet) Ack() bool        { return p[1]&0xf0 == CmdAck }
func (p Packet) Completion() bool { return p[1]&0xf0 == CmdCompletion }
func (p Packet) Type() Type       { return Type(0x0f & p[1]) }
func (p Packet) Payload() []byte  { return p[2 : len(p)-1] }
func (p Packet) Len() int         { return len(p) }

func (p Packet) Error() (err error) {
	if p[1]&0xf0 == CmdErr {
		switch p[2] {
		case SyntaxError:
			err = ErrSyntax
		case CommandBufferFull:
			err = ErrBufferFull
		case CommandCanceled:
			err = ErrCanceled
		case NoSocket:
			err = ErrNoSocket
		case CommandNotExecutable:
			err = ErrNotExecutable
		default:
			err = ErrUnknown
		}
	}
	return err
}

func (p Packet) Value(offset int, width int) (val int) {
	payload := p.Payload()[offset : offset+width]
	for len(payload) > 0 {
		val <<= 4
		val |= int(0x0f & payload[0])
		payload = payload[1:]
	}
	return val
}

type Info struct {
	PanPos  int
	TiltPos int
	ZoomPos int
}

type Camera interface {
	Query() (Info, error)
}

func New(host string) (cam Camera, err error) {
	cam = &camera{
		host: host,
	}
	return cam, err
}

type camera struct {
	host string
}

func (cam *camera) Connect() (Conn, error) {
	// TODO: MAGIC NUMBERS!! (time.Second*10)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", cam.host, ViscaPort), time.Second*10)
	if err == nil {
		return &netConn{Conn: &connectionWithDeadline{conn, time.Second * 10}}, nil
	}
	return nil, err
}

type Conn interface {
	SendCommand(Command, []byte) (Packet, error)
	SendInquiry(Command) (Packet, error)
}

type netConn struct {
	net.Conn
	reader *bufio.Reader
}

func (conn *netConn) write(pkt []byte) error {
	_, err := conn.Conn.Write(pkt)
	return err
}

func (conn *netConn) readPacket() (pkt Packet, err error) {
	if conn.reader == nil {
		conn.reader = bufio.NewReader(conn.Conn)
	}
	pkt = make(Packet, 0)
	var b byte
	for b, err = conn.reader.ReadByte(); err == nil && b != TerminatorByte; b, err = conn.reader.ReadByte() {
		pkt = append(pkt, b)
	}
	if err == nil {
		pkt = append(pkt, 0xff)
	}
	return pkt, err
}

func (conn *netConn) read() (Packet, error) {
	pkt, err := conn.readPacket()
	for err == nil && pkt.Ack() {
		pkt, err = conn.readPacket()
	}
	return pkt, err
}

type connectionWithDeadline struct {
	net.Conn
	timeout time.Duration
}

func (conn *connectionWithDeadline) Read(b []byte) (n int, err error) {
	err = conn.SetReadDeadline(time.Now().Add(conn.timeout))
	defer conn.SetReadDeadline(time.Time{})
	if err == nil {
		n, err = conn.Conn.Read(b)
	}
	return
}

func (conn *connectionWithDeadline) Write(b []byte) (n int, err error) {
	err = conn.SetWriteDeadline(time.Now().Add(conn.timeout))
	defer conn.SetWriteDeadline(time.Time{})
	if err == nil {
		n, err = conn.Conn.Write(b)
	}
	return
}

func (conn *netConn) send(buf []byte) (pkt Packet, err error) {
	_, err = conn.Write(buf)
	if err == nil {
		pkt, err = conn.read()
	}
	return pkt, err
}

func (conn *netConn) SendCommand(cmd Command, payload []byte) (Packet, error) {
	pkt := append([]byte{HeaderByte, CommandByte, cmd[0], cmd[1]}, payload...)
	pkt = append(pkt, TerminatorByte)
	return conn.send(pkt)
}

func (conn *netConn) SendInquiry(cmd Command) (Packet, error) {
	return conn.send([]byte{HeaderByte, InquiryByte, cmd[0], cmd[1], TerminatorByte})
}

func (cam *camera) Query() (info Info, err error) {
	conn, err := cam.Connect()
	if err == nil {

		pkt, err := conn.SendInquiry(InqZoomPos)
		if err == nil {
			info.ZoomPos = pkt.Value(0, 4)

			pkt, err = conn.SendInquiry(InqPanTiltPos)
			if err == nil {
				info.PanPos = pkt.Value(0, 4)
				info.TiltPos = pkt.Value(4, 4)
			}
		}
	}
	return
}
