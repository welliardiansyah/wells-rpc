package codecgenerated

import (
	"errors"

	wellsrpc "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc"
)

const (
	maxMessageSize = 4 * 1024 * 1024 // 4MB safety guard
)

// =======================
// SensorReading
// =======================
type SensorReading struct {
	Timestamp   int64
	Temperature float32
	Humidity    float32
	Payload     []byte
}

// Marker untuk codegen / tooling / future reflection
func (*SensorReading) WelliMessageName() string {
	return "SensorReading"
}

func (s *SensorReading) MarshalWells() []byte {
	buf := wellsrpc.GetBuffer()
	defer wellsrpc.PutBuffer(buf)
	b := (*buf)[:0]

	// field 1: timestamp (varint + zigzag)
	b = append(b, 0x08)
	b = append(b, wellsrpc.EncodeVarint(wellsrpc.ZigzagEncode(s.Timestamp))...)

	// field 2: temperature (fixed32)
	b = append(b, 0x15)
	wellsrpc.WriteFloat32LE(&b, s.Temperature)

	// field 3: humidity (fixed32)
	b = append(b, 0x1D)
	wellsrpc.WriteFloat32LE(&b, s.Humidity)

	// field 4: payload (bytes)
	if len(s.Payload) > 0 {
		b = append(b, 0x22)
		b = append(b, wellsrpc.EncodeVarint(uint64(len(s.Payload)))...)
		b = append(b, s.Payload...)
	}

	out := make([]byte, len(b))
	copy(out, b)
	return out
}

func (s *SensorReading) UnmarshalWells(b []byte) error {
	if len(b) > maxMessageSize {
		return errors.New("sensor reading: message too large")
	}

	var i int
	for i < len(b) {
		tag := b[i]
		i++

		fieldNum := int(tag >> 3)
		wireType := int(tag & 0x7)

		switch fieldNum {

		// timestamp
		case 1:
			if wireType != 0 {
				return errors.New("sensor reading: invalid wire type for timestamp")
			}
			val, n := wellsrpc.DecodeVarint(b[i:])
			if n == 0 {
				return errors.New("sensor reading: invalid timestamp varint")
			}
			s.Timestamp = wellsrpc.ZigzagDecode(val)
			i += n

		// temperature
		case 2:
			if wireType != 5 || i+4 > len(b) {
				return errors.New("sensor reading: temperature truncated")
			}
			s.Temperature = wellsrpc.ReadFloat32LE(b[i : i+4])
			i += 4

		// humidity
		case 3:
			if wireType != 5 || i+4 > len(b) {
				return errors.New("sensor reading: humidity truncated")
			}
			s.Humidity = wellsrpc.ReadFloat32LE(b[i : i+4])
			i += 4

		// payload
		case 4:
			if wireType != 2 {
				return errors.New("sensor reading: invalid wire type for payload")
			}
			l, n := wellsrpc.DecodeVarint(b[i:])
			if n == 0 || int(l) > maxMessageSize {
				return errors.New("sensor reading: invalid payload length")
			}
			i += n
			if i+int(l) > len(b) {
				return errors.New("sensor reading: payload truncated")
			}
			s.Payload = append([]byte(nil), b[i:i+int(l)]...)
			i += int(l)

		// unknown field → skip safely
		default:
			switch wireType {
			case 0:
				_, n := wellsrpc.DecodeVarint(b[i:])
				if n == 0 {
					return errors.New("sensor reading: invalid skip varint")
				}
				i += n
			case 2:
				l, n := wellsrpc.DecodeVarint(b[i:])
				if n == 0 {
					return errors.New("sensor reading: invalid skip length")
				}
				i += n + int(l)
			case 5:
				i += 4
			case 1:
				i += 8
			default:
				return errors.New("sensor reading: unknown wire type")
			}
		}
	}
	return nil
}

// =======================
// Ack
// =======================
type Ack struct {
	Success bool
}

func (*Ack) WelliMessageName() string {
	return "Ack"
}

func (a *Ack) MarshalWells() []byte {
	buf := wellsrpc.GetBuffer()
	defer wellsrpc.PutBuffer(buf)
	b := (*buf)[:0]

	// field 1: success (bool → varint)
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

func (a *Ack) UnmarshalWells(b []byte) error {
	if len(b) > maxMessageSize {
		return errors.New("ack: message too large")
	}

	var i int
	for i < len(b) {
		tag := b[i]
		i++

		fieldNum := int(tag >> 3)
		wireType := int(tag & 0x7)

		switch fieldNum {

		case 1:
			if wireType != 0 {
				return errors.New("ack: invalid wire type")
			}
			val, n := wellsrpc.DecodeVarint(b[i:])
			if n == 0 {
				return errors.New("ack: invalid varint")
			}
			a.Success = val != 0
			i += n

		default:
			// skip unknown
			switch wireType {
			case 0:
				_, n := wellsrpc.DecodeVarint(b[i:])
				if n == 0 {
					return errors.New("ack: invalid skip")
				}
				i += n
			case 2:
				l, n := wellsrpc.DecodeVarint(b[i:])
				if n == 0 {
					return errors.New("ack: invalid skip length")
				}
				i += n + int(l)
			default:
				return errors.New("ack: unknown wire type")
			}
		}
	}
	return nil
}
