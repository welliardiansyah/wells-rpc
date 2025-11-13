package codecgenerated

import (
	"errors"

	wellsrpc "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc"
)

type SensorReading struct {
	Timestamp   int64
	Temperature float32
	Humidity    float32
	Payload     []byte
}

type Ack struct {
	Success bool
}

func (s *SensorReading) MarshalWelli() []byte {
	buf := wellsrpc.GetBuffer()
	defer wellsrpc.PutBuffer(buf)
	b := *buf

	b = append(b, 0x08)
	b = append(b, wellsrpc.EncodeVarint(wellsrpc.ZigzagEncode(s.Timestamp))...)

	b = append(b, 0x15)
	wellsrpc.WriteFloat32LE(&b, s.Temperature)

	b = append(b, 0x1D)
	wellsrpc.WriteFloat32LE(&b, s.Humidity)

	if len(s.Payload) > 0 {
		b = append(b, 0x22)
		b = append(b, wellsrpc.EncodeVarint(uint64(len(s.Payload)))...)
		b = append(b, s.Payload...)
	}

	out := make([]byte, len(b))
	copy(out, b)
	return out
}

func (s *SensorReading) UnmarshalWelli(b []byte) error {
	var i int
	for i < len(b) {
		tag := b[i]
		i++
		fieldNum := int(tag >> 3)
		wireType := int(tag & 0x7)
		switch fieldNum {
		case 1:
			val, n := wellsrpc.DecodeVarint(b[i:])
			if n == 0 {
				return errors.New("invalid varint timestamp")
			}
			s.Timestamp = wellsrpc.ZigzagDecode(val)
			i += n
		case 2:
			if i+4 > len(b) {
				return errors.New("temperature truncated")
			}
			s.Temperature = wellsrpc.ReadFloat32LE(b[i : i+4])
			i += 4
		case 3:
			if i+4 > len(b) {
				return errors.New("humidity truncated")
			}
			s.Humidity = wellsrpc.ReadFloat32LE(b[i : i+4])
			i += 4
		case 4:
			length, n := wellsrpc.DecodeVarint(b[i:])
			if n == 0 {
				return errors.New("invalid payload length")
			}
			i += n
			if i+int(length) > len(b) {
				return errors.New("payload too short")
			}
			s.Payload = append([]byte(nil), b[i:i+int(length)]...)
			i += int(length)
		default:
			switch wireType {
			case 0:
				_, n := wellsrpc.DecodeVarint(b[i:])
				if n == 0 {
					return errors.New("invalid varint in skip")
				}
				i += n
			case 2:
				l, n := wellsrpc.DecodeVarint(b[i:])
				if n == 0 {
					return errors.New("invalid length in skip")
				}
				i += n + int(l)
			case 5:
				i += 4
			case 1:
				i += 8
			default:
				return errors.New("unknown wire type")
			}
		}
	}
	return nil
}

func (a *Ack) MarshalWelli() []byte {
	buf := wellsrpc.GetBuffer()
	defer wellsrpc.PutBuffer(buf)
	b := *buf
	b = append(b, 0x08)
	if a.Success {
		b = append(b, 1)
	} else {
		b = append(b, 0)
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

func (a *Ack) UnmarshalWelli(b []byte) error {
	var i int
	for i < len(b) {
		tag := b[i]
		i++
		fieldNum := int(tag >> 3)
		switch fieldNum {
		case 1:
			val, n := wellsrpc.DecodeVarint(b[i:])
			if n == 0 {
				return errors.New("invalid varint for ack")
			}
			a.Success = val != 0
			i += n
		default:
			_, n := wellsrpc.DecodeVarint(b[i:])
			if n == 0 {
				return errors.New("invalid skip")
			}
			i += n
		}
	}
	return nil
}
