package wellsrpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Handler func(ctx context.Context, payload []byte) ([]byte, error)
type StreamHandler func(ctx context.Context, s *Stream) error

type RPCServer struct {
	handlers     map[string]Handler
	streams      map[string]StreamHandler
	handlersLock sync.RWMutex
	streamsLock  sync.RWMutex

	unaryInterceptors []UnaryServerInterceptor
	tlsConfig         *tls.Config
}

func NewRPCServer() *RPCServer {
	return &RPCServer{
		handlers: make(map[string]Handler),
		streams:  make(map[string]StreamHandler),
	}
}

func (s *RPCServer) WithTLS(cfg *tls.Config) {
	s.tlsConfig = cfg
}

func (s *RPCServer) Register(method string, h Handler) {
	s.handlersLock.Lock()
	s.handlers[method] = h
	s.handlersLock.Unlock()
}

func (s *RPCServer) RegisterStream(method string, h StreamHandler) {
	s.streamsLock.Lock()
	s.streams[method] = h
	s.streamsLock.Unlock()
}

func (s *RPCServer) UseUnaryInterceptor(i UnaryServerInterceptor) {
	s.unaryInterceptors = append(s.unaryInterceptors, i)
}

func (s *RPCServer) Serve(addr string) error {
	var ln net.Listener
	var err error
	if s.tlsConfig != nil {
		ln, err = tls.Listen("tcp", addr, s.tlsConfig)
	} else {
		ln, err = net.Listen("tcp", addr)
	}
	if err != nil {
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.serveConn(conn)
	}
}

func (s *RPCServer) serveConn(conn net.Conn) {
	defer conn.Close()
	streamMap := make(map[uint32]*Stream)
	var smu sync.Mutex
	send := func(f *Frame) error { return WriteFrame(conn, f) }

	for {
		frame, err := ReadFrame(conn)
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Println("read frame err:", err)
			return
		}
		switch frame.Type {
		case FrameTypeRequest:
			go s.handleUnary(conn, frame)
		case FrameTypeStreamOpen:
			s.streamsLock.RLock()
			sh, ok := s.streams[frame.Method]
			s.streamsLock.RUnlock()
			if !ok {
				_ = WriteFrame(conn, &Frame{Type: FrameTypeError, StreamID: frame.StreamID, Payload: []byte("stream handler not found")})
				continue
			}
			stream := newStream(frame.StreamID, func(data []byte) error {
				return send(&Frame{Type: FrameTypeStreamData, StreamID: frame.StreamID, Payload: data})
			})
			smu.Lock()
			streamMap[frame.StreamID] = stream
			smu.Unlock()
			go func() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				_ = sh(ctx, stream)
				_ = WriteFrame(conn, &Frame{Type: FrameTypeStreamClose, StreamID: frame.StreamID})
				smu.Lock()
				if st, ok := streamMap[frame.StreamID]; ok {
					st.Close()
					delete(streamMap, frame.StreamID)
				}
				smu.Unlock()
			}()
		case FrameTypeStreamData:
			smu.Lock()
			st, ok := streamMap[frame.StreamID]
			smu.Unlock()
			if ok {
				select {
				case st.recvCh <- frame.Payload:
				default:
					// drop to avoid blocking; you might add backpressure
				}
			}
		case FrameTypeStreamClose:
			smu.Lock()
			if st, ok := streamMap[frame.StreamID]; ok {
				st.Close()
				delete(streamMap, frame.StreamID)
			}
			smu.Unlock()
		case FrameTypePing:
			_ = WriteFrame(conn, &Frame{Type: FrameTypePong, StreamID: frame.StreamID})
		}
	}
}

func (s *RPCServer) handleUnary(conn net.Conn, frame *Frame) {
	method := frame.Method
	s.handlersLock.RLock()
	h, ok := s.handlers[method]
	s.handlersLock.RUnlock()
	if !ok {
		_ = WriteFrame(conn, &Frame{Type: FrameTypeError, StreamID: frame.StreamID, Payload: []byte("method not found: " + method)})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	exec := func(ctx context.Context, payload []byte) ([]byte, error) {
		return h(ctx, payload)
	}
	var chained Handler = exec
	for i := len(s.unaryInterceptors) - 1; i >= 0; i-- {
		inter := s.unaryInterceptors[i]
		next := chained
		chained = func(ctx context.Context, payload []byte) ([]byte, error) {
			return inter(ctx, payload, func(c context.Context, p []byte) ([]byte, error) {
				return next(c, p)
			})
		}
	}

	resp, err := chained(ctx, frame.Payload)
	if err != nil {
		_ = WriteFrame(conn, &Frame{Type: FrameTypeError, StreamID: frame.StreamID, Payload: []byte(err.Error())})
		return
	}
	_ = WriteFrame(conn, &Frame{Type: FrameTypeResponse, StreamID: frame.StreamID, Payload: resp})
}
