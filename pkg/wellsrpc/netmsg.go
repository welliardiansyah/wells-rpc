package wellsrpc

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"time"
)

const (
	readTimeout  = 5 * time.Second
	writeTimeout = 5 * time.Second
	maxMsgSize   = 64 * 1024 * 1024
)

func SendMessage(conn net.Conn, msg WelliMarshaller) error {
	data := msg.MarshalWelli()
	header := [4]byte{}
	binary.LittleEndian.PutUint32(header[:], uint32(len(data)))

	if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return err
	}
	if _, err := conn.Write(header[:]); err != nil {
		return err
	}
	_, err := conn.Write(data)
	return err
}

func ReceiveMessage(conn net.Conn, msg WelliMarshaller) error {
	header := [4]byte{}
	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		return err
	}
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return err
	}
	size := binary.LittleEndian.Uint32(header[:])
	if size > maxMsgSize {
		return errors.New("message too large")
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(conn, data); err != nil {
		return err
	}
	return Unmarshal(msg, data)
}
