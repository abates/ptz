package visca

import (
	"bytes"
	"reflect"
	"testing"
)

func TestPacket_Error(t *testing.T) {
	tests := []struct {
		name    string
		p       Packet
		wantErr error
	}{
		{"MessageLenError", Packet{0x90, 0x60, MessageLenError}, ErrMessageLen},
		{"SyntaxError", Packet{0x90, 0x60, SyntaxError}, ErrSyntax},
		{"CommandBufferFull", Packet{0x90, 0x60, CommandBufferFull}, ErrBufferFull},
		{"CommandCanceled", Packet{0x90, 0x60, CommandCanceled}, ErrCanceled},
		{"NoSocket", Packet{0x90, 0x60, NoSocket}, ErrNoSocket},
		{"CommandNotExecutable", Packet{0x90, 0x60, CommandNotExecutable}, ErrNotExecutable},
		{"default", Packet{0x90, 0x60, 0xff}, ErrUnknown},
		{"no error", Packet{0x90, 0x40, 0xff}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotErr := tt.p.Error(); tt.wantErr != gotErr {
				t.Errorf("Packet.Error() error = %v, wantErr %v", gotErr, tt.wantErr)
			}
		})
	}
}

func TestPacket_Header(t *testing.T) {
	tests := []struct {
		name string
		p    Packet
		want Header
	}{
		{"one", Packet{0x01, 0x00, 0xff}, Header(0x01)},
		{"two", Packet{0x02, 0x00, 0xff}, Header(0x02)},
		{"three", Packet{0x03, 0x00, 0xff}, Header(0x03)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Header(); got != tt.want {
				t.Errorf("Packet.Header() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPacket_Ack(t *testing.T) {
	tests := []struct {
		name string
		p    Packet
		want bool
	}{
		{"ACK", Packet{0x01, 0x4f, 0xff}, true},
		{"No ACK", Packet{0x02, 0xf5, 0xff}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Ack(); got != tt.want {
				t.Errorf("Packet.Ack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPacket_Completion(t *testing.T) {
	tests := []struct {
		name string
		p    Packet
		want bool
	}{
		{"Completion", Packet{0x01, 0x5f, 0xff}, true},
		{"No Completion", Packet{0x02, 0xf4, 0xff}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Completion(); got != tt.want {
				t.Errorf("Packet.Completion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPacket_Type(t *testing.T) {
	tests := []struct {
		name string
		p    Packet
		want Type
	}{
		{"Command", Packet{0x81, 0x01, 0x01, 0x02, 0x0a, 0x0b, 0x0c, 0xff}, Cmd},
		{"Inquiry", Packet{0x81, 0x09, 0x01, 0x02, 0xff}, Inq},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Type(); got != tt.want {
				t.Errorf("Packet.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPacket_Payload(t *testing.T) {
	tests := []struct {
		name string
		p    Packet
		want []byte
	}{
		{"Command", Packet{0x81, 0x01, 0x01, 0x02, 0x0a, 0x0b, 0x0c, 0xff}, []byte{0x01, 0x02, 0x0a, 0x0b, 0x0c}},
		{"Inquiry", Packet{0x81, 0x09, 0x0a, 0x0b, 0x0c, 0xff}, []byte{0x0a, 0x0b, 0x0c}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Payload(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Packet.Payload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPacket_Len(t *testing.T) {
	tests := []struct {
		name string
		p    Packet
		want int
	}{
		{"Command", Packet{0x81, 0x01, 0x01, 0x02, 0x0a, 0x0b, 0x0c, 0xff}, 8},
		{"Inquiry", Packet{0x81, 0x09, 0x0a, 0x0b, 0x0c, 0xff}, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Len(); got != tt.want {
				t.Errorf("Packet.Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPacket_Value(t *testing.T) {
	type args struct {
		offset int
		width  int
	}

	tests := []struct {
		name    string
		p       Packet
		args    args
		wantVal int
	}{
		{"offset 0", Packet{0x90, 0x50, 0x03, 0x00, 0x0a, 0x02, 0xff}, args{0, 4}, 0x30a2},
		{"offset 4", Packet{0x90, 0x50, 0x00, 0x00, 0x00, 0x0a, 0x0f, 0x0f, 0x06, 0x00, 0xff}, args{4, 4}, 0xff60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotVal := tt.p.Value(tt.args.offset, tt.args.width); gotVal != tt.wantVal {
				t.Errorf("Packet.Value() = %v, want %v", gotVal, tt.wantVal)
			}
		})
	}
}

func TestReadPacket(t *testing.T) {
	tests := []struct {
		name    string
		buf     []byte
		wantPkt Packet
		wantErr bool
	}{
		{"single packet", []byte{0x90, 0x50, 0x03, 0x00, 0x0a, 0x02, 0xff}, Packet{0x90, 0x50, 0x03, 0x00, 0x0a, 0x02, 0xff}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw := New(bytes.NewBuffer(tt.buf), nil).(*conn)
			gotPkt, err := rw.readPacket()
			if (err != nil) != tt.wantErr {
				t.Errorf("conn.readPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotPkt, tt.wantPkt) {
				t.Errorf("conn.readPacket() = %v, want %v", gotPkt, tt.wantPkt)
			}
		})
	}
}

func TestRead(t *testing.T) {
	tests := []struct {
		name    string
		buf     []byte
		want    Packet
		wantErr bool
	}{
		{"single packet", []byte{0x90, 0x50, 0x03, 0x00, 0x0a, 0x02, 0xff}, Packet{0x90, 0x50, 0x03, 0x00, 0x0a, 0x02, 0xff}, false},
		{"ack followed by single packet", []byte{0x90, 0x40, 0xff, 0x90, 0x50, 0x03, 0x00, 0x0a, 0x02, 0xff}, Packet{0x90, 0x50, 0x03, 0x00, 0x0a, 0x02, 0xff}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw := New(bytes.NewBuffer(tt.buf), nil).(*conn)
			got, err := rw.read()
			if (err != nil) != tt.wantErr {
				t.Errorf("conn.read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("conn.read() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Testsend(t *testing.T) {
	type args struct {
		buf []byte
	}

	tests := []struct {
		name    string
		buf     []byte
		args    args
		wantPkt Packet
		wantErr bool
	}{
		{"", []byte{0x90, 0x50, 0x03, 0x00, 0x0a, 0x02, 0xff}, args{[]byte{0x81, 0x09, 0x04, 0x47, 0xff}}, Packet{0x90, 0x50, 0x03, 0x00, 0x0a, 0x02, 0xff}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bytes.NewBuffer(nil)
			rw := New(bytes.NewBuffer(tt.buf), got).(*conn)
			gotPkt, err := rw.send(tt.args.buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("conn.send() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(gotPkt, tt.wantPkt) {
				t.Errorf("conn.send() = %v, want %v", gotPkt, tt.wantPkt)
			}

			if !reflect.DeepEqual(got.Bytes(), tt.args.buf) {
				t.Errorf("RWCoonn.send() sent %v, want %v", got.Bytes(), tt.args.buf)
			}
		})
	}
}

func TestSendCommand(t *testing.T) {
	type args struct {
		cmd     Command
		payload []byte
	}
	tests := []struct {
		name     string
		cmd      Command
		payload  []byte
		buf      []byte
		wantSent []byte
		wantRcv  Packet
		wantErr  bool
	}{
		{"", Command{0x01, 0x02}, []byte{1, 2, 3, 4}, []byte{0x90, 0x50, 0xff}, []byte{0x81, 0x01, 0x01, 0x02, 1, 2, 3, 4, 0xff}, Packet{0x90, 0x50, 0xff}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBuf := bytes.NewBuffer(nil)
			rw := New(bytes.NewBuffer(tt.buf), gotBuf).(*conn)
			got, err := rw.SendCommand(tt.cmd, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("conn.SendCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.wantRcv) {
				t.Errorf("conn.SendCommand() = %v, want %v", got, tt.wantRcv)
			}
			if !reflect.DeepEqual(gotBuf.Bytes(), tt.wantSent) {
				t.Errorf("RWCoonn.send() sent %v, want %v", gotBuf.Bytes(), tt.wantSent)
			}
		})
	}
}

func TestSendInquiry(t *testing.T) {
	tests := []struct {
		name     string
		cmd      Command
		buf      []byte
		wantSent []byte
		wantRcv  Packet
		wantErr  bool
	}{
		{"", InqZoomPos, []byte{0x90, 0x50, 0x03, 0x00, 0x0a, 0x02, 0xff}, []byte{0x81, 0x09, 0x04, 0x47, 0xff}, Packet{0x90, 0x50, 0x03, 0x00, 0x0a, 0x02, 0xff}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBuf := bytes.NewBuffer(nil)
			rw := New(bytes.NewBuffer(tt.buf), gotBuf).(*conn)
			got, err := rw.SendInquiry(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("conn.SendInquiry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.wantRcv) {
				t.Errorf("conn.SendInquiry() = %v, want %v", got, tt.wantRcv)
			}
			if !reflect.DeepEqual(gotBuf.Bytes(), tt.wantSent) {
				t.Errorf("RWCoonn.send() sent %v, want %v", gotBuf.Bytes(), tt.wantSent)
			}
		})
	}
}
