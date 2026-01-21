package wellsrpc

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	FrameTypeRequest     = 0x00
	FrameTypeResponse    = 0x01
	FrameTypeError       = 0x02
	FrameTypeStreamOpen  = 0x10
	FrameTypeStreamData  = 0x11
	FrameTypeStreamClose = 0x12
	FrameTypePing        = 0xFE
	FrameTypePong        = 0xFF
)

type Frame struct {
	Type     byte
	StreamID uint32
	Method   string
	Metadata map[string]string
	Payload  []byte
}

func WriteFrame(w io.Writer, f *Frame) error {
	if f.Metadata == nil {
		f.Metadata = map[string]string{}
	}

	bufp := GetBuffer()
	defer PutBuffer(bufp)
	buf := (*bufp)[:0]

	// reserve length
	buf = append(buf, 0, 0, 0, 0)

	// frame type
	buf = append(buf, f.Type)

	// stream id
	var tmp4 [4]byte
	binary.LittleEndian.PutUint32(tmp4[:], f.StreamID)
	buf = append(buf, tmp4[:]...)

	// method
	methodLen := len(f.Method)
	buf = append(buf, byte(methodLen))
	if methodLen > 0 {
		buf = append(buf, f.Method...)
	}

	// metadata
	metaCount := len(f.Metadata)
	if metaCount > 255 {
		return errors.New("too many metadata entries")
	}
	buf = append(buf, byte(metaCount))

	for k, v := range f.Metadata {
		if len(k) > 255 || len(v) > 255 {
			return errors.New("metadata key/value too long")
		}
		buf = append(buf, byte(len(k)))
		buf = append(buf, k...)
		buf = append(buf, byte(len(v)))
		buf = append(buf, v...)
	}

	// payload
	if len(f.Payload) > 0 {
		buf = append(buf, f.Payload...)
	}

	// write total length (exclude first 4 bytes)
	binary.LittleEndian.PutUint32(buf[:4], uint32(len(buf)-4))

	_, err := w.Write(buf)
	return err
}

func ReadFrame(r io.Reader) (*Frame, error) {
	// read frame length
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, err
	}

	totalLen := binary.LittleEndian.Uint32(hdr)
	if totalLen < 6 {
		return nil, errors.New("frame too small")
	}

	body := make([]byte, totalLen)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}

	idx := 0

	// frame type
	ft := body[idx]
	idx++

	// stream id
	if idx+4 > len(body) {
		return nil, errors.New("invalid stream id")
	}
	streamID := binary.LittleEndian.Uint32(body[idx : idx+4])
	idx += 4

	// method
	if idx >= len(body) {
		return nil, errors.New("invalid method length")
	}
	methodLen := int(body[idx])
	idx++

	if idx+methodLen > len(body) {
		return nil, errors.New("invalid method data")
	}
	method := ""
	if methodLen > 0 {
		method = string(body[idx : idx+methodLen])
		idx += methodLen
	}

	// metadata
	if idx >= len(body) {
		return nil, errors.New("invalid metadata count")
	}
	metaCount := int(body[idx])
	idx++

	meta := make(map[string]string, metaCount)
	for i := 0; i < metaCount; i++ {
		if idx >= len(body) {
			return nil, errors.New("invalid metadata key length")
		}
		kl := int(body[idx])
		idx++
		if idx+kl > len(body) {
			return nil, errors.New("invalid metadata key")
		}
		key := string(body[idx : idx+kl])
		idx += kl

		if idx >= len(body) {
			return nil, errors.New("invalid metadata value length")
		}
		vl := int(body[idx])
		idx++
		if idx+vl > len(body) {
			return nil, errors.New("invalid metadata value")
		}
		val := string(body[idx : idx+vl])
		idx += vl

		meta[key] = val
	}

	// payload
	payload := body[idx:]

	return &Frame{
		Type:     ft,
		StreamID: streamID,
		Method:   method,
		Metadata: meta,
		Payload:  payload,
	}, nil
}
