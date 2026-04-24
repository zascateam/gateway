package tunnel

import (
	"encoding/binary"
	"io"
)

const (
	ChannelRDP        byte = 0x01
	ChannelWinRM      byte = 0x02
	ChannelRemoteExec byte = 0x03
	ChannelControl    byte = 0xFF
)

type Frame struct {
	Channel byte
	Payload []byte
}

func (f Frame) Marshal() ([]byte, error) {
	length := uint16(len(f.Payload))
	buf := make([]byte, 3+length)
	binary.BigEndian.PutUint16(buf[0:2], length)
	buf[2] = f.Channel
	copy(buf[3:], f.Payload)
	return buf, nil
}

func UnmarshalFrame(r io.Reader) (Frame, error) {
	var header [3]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Frame{}, err
	}

	length := binary.BigEndian.Uint16(header[0:2])
	channel := header[2]

	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return Frame{}, err
		}
	}

	return Frame{
		Channel: channel,
		Payload: payload,
	}, nil
}
