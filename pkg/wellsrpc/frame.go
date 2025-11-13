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
	Payload  []byte
}

func WriteFrame(w io.Writer, f *Frame) error {
	methodLen := len(f.Method)
	headerLen := 1 + 4 + 1 + methodLen
	totalLen := uint32(headerLen + len(f.Payload))

	bufp := GetBuffer()
	defer PutBuffer(bufp)
	buf := *bufp

	buf = append(buf, 0, 0, 0, 0)
	buf = append(buf, f.Type)
	var tmp4 [4]byte
	binary.LittleEndian.PutUint32(tmp4[:], f.StreamID)
	buf = append(buf, tmp4[:]...)
	buf = append(buf, byte(methodLen))
	if methodLen > 0 {
		buf = append(buf, f.Method...)
	}
	if len(f.Payload) > 0 {
		buf = append(buf, f.Payload...)
	}

	binary.LittleEndian.PutUint32(buf[:4], totalLen)
	_, err := w.Write(buf)
	return err
}

func ReadFrame(r io.Reader) (*Frame, error) {
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
	ft := body[idx]
	idx++
	streamID := binary.LittleEndian.Uint32(body[idx : idx+4])
	idx += 4
	methodLen := int(body[idx])
	idx++
	var method string
	if methodLen > 0 {
		if idx+methodLen > len(body) {
			return nil, errors.New("invalid method length")
		}
		method = string(body[idx : idx+methodLen])
		idx += methodLen
	}
	payload := body[idx:]
	return &Frame{Type: ft, StreamID: streamID, Method: method, Payload: payload}, nil
}
