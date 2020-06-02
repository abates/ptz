package visca

import (
	"bufio"
	"errors"
	"io"
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
	Cmd Type = 0x01
	Inq Type = 0x09
)

const (
	CmdCompletion byte = 0x50
	CmdAck        byte = 0x40
	CmdErr        byte = 0x60

	MessageLenError      byte = 0x01
	SyntaxError               = 0x02
	CommandBufferFull         = 0x03
	CommandCanceled           = 0x04
	NoSocket                  = 0x05
	CommandNotExecutable      = 0x41
)

var (
	ErrMessageLen    = errors.New("Incorrect message length")
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
		case MessageLenError:
			err = ErrMessageLen
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

type Conn interface {
	SendCommand(Command, []byte) (Packet, error)
	SendInquiry(Command) (Packet, error)
}

type conn struct {
	*bufio.Reader
	io.Writer
}

func New(reader io.Reader, writer io.Writer) Conn {
	return &conn{bufio.NewReader(reader), writer}
}

func (rw *conn) readPacket() (pkt Packet, err error) {
	pkt = make(Packet, 0)
	var b byte
	for b, err = rw.ReadByte(); err == nil && b != TerminatorByte; b, err = rw.ReadByte() {
		pkt = append(pkt, b)
	}

	if err == nil {
		pkt = append(pkt, 0xff)
	}
	return pkt, err
}

func (rw *conn) read() (Packet, error) {
	pkt, err := rw.readPacket()
	for err == nil && pkt.Ack() {
		pkt, err = rw.readPacket()
	}
	return pkt, err
}

func (rw *conn) send(buf []byte) (pkt Packet, err error) {
	_, err = rw.Write(buf)
	if err == nil {
		pkt, err = rw.read()
	}
	return pkt, err
}

func (rw *conn) SendCommand(cmd Command, payload []byte) (Packet, error) {
	pkt := append([]byte{HeaderByte, CommandByte, cmd[0], cmd[1]}, payload...)
	pkt = append(pkt, TerminatorByte)
	return rw.send(pkt)
}

func (rw *conn) SendInquiry(cmd Command) (Packet, error) {
	return rw.send([]byte{HeaderByte, InquiryByte, cmd[0], cmd[1], TerminatorByte})
}
