package wellsrpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync"
)

// =======================
// Handler definitions
// =======================
// Handler bisnis TETAP payload-based agar codegen & user code
// tidak perlu berubah.
type Handler func(ctx context.Context, payload []byte) ([]byte, error)
type StreamHandler func(ctx context.Context, s *Stream) error

// =======================
// RPC Server
// =======================
type RPCServer struct {
	handlers map[string]Handler
	streams  map[string]StreamHandler

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

// =======================
// Serve
// =======================
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

// =======================
// Connection loop
// =======================
func (s *RPCServer) serveConn(conn net.Conn) {
	defer conn.Close()

	streamMap := make(map[uint32]*Stream)
	var smu sync.Mutex

	send := func(f *Frame) error {
		return WriteFrame(conn, f)
	}

	for {
		frame, err := ReadFrame(conn)
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Println("read frame error:", err)
			return
		}

		switch frame.Type {

		// -----------------------
		// Unary request
		// -----------------------
		case FrameTypeRequest:
			go s.handleUnary(conn, frame)

		// -----------------------
		// Stream open
		// -----------------------
		case FrameTypeStreamOpen:
			s.streamsLock.RLock()
			sh, ok := s.streams[frame.Method]
			s.streamsLock.RUnlock()

			if !ok {
				_ = send(&Frame{
					Type:     FrameTypeError,
					StreamID: frame.StreamID,
					Method:   frame.Method,
					Metadata: frame.Metadata,
					Payload:  []byte("stream handler not found"),
				})
				continue
			}

			stream := newStream(frame.StreamID, func(data []byte) error {
				return send(&Frame{
					Type:     FrameTypeStreamData,
					StreamID: frame.StreamID,
					Method:   frame.Method,
					Metadata: frame.Metadata,
					Payload:  data,
				})
			})

			smu.Lock()
			streamMap[frame.StreamID] = stream
			smu.Unlock()

			go func(f *Frame) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				_ = sh(ctx, stream)

				_ = send(&Frame{
					Type:     FrameTypeStreamClose,
					StreamID: f.StreamID,
					Method:   f.Method,
					Metadata: f.Metadata,
				})

				smu.Lock()
				if st, ok := streamMap[f.StreamID]; ok {
					st.Close()
					delete(streamMap, f.StreamID)
				}
				smu.Unlock()
			}(frame)

		// -----------------------
		// Stream data
		// -----------------------
		case FrameTypeStreamData:
			smu.Lock()
			st, ok := streamMap[frame.StreamID]
			smu.Unlock()
			if ok {
				select {
				case st.recvCh <- frame.Payload:
				default:
				}
			}
		}
	}
}

// =======================
// Unary handling (INTERCEPTOR-AWARE)
// =======================
func (s *RPCServer) handleUnary(conn net.Conn, f *Frame) {
	s.handlersLock.RLock()
	h, ok := s.handlers[f.Method]
	s.handlersLock.RUnlock()

	if !ok {
		_ = WriteFrame(conn, &Frame{
			Type:     FrameTypeError,
			StreamID: f.StreamID,
			Method:   f.Method,
			Metadata: f.Metadata,
			Payload:  []byte("handler not found"),
		})
		return
	}

	// -----------------------
	// Base invoke (business handler)
	// -----------------------
	invoke := func(ctx context.Context, fr *Frame) (*Frame, error) {
		out, err := h(ctx, fr.Payload)
		if err != nil {
			return nil, err
		}
		return &Frame{
			Type:     FrameTypeResponse,
			StreamID: fr.StreamID,
			Method:   fr.Method,
			Metadata: fr.Metadata,
			Payload:  out,
		}, nil
	}

	// -----------------------
	// Build interceptor chain
	// -----------------------
	ctx := context.Background()
	chained := invoke

	for i := len(s.unaryInterceptors) - 1; i >= 0; i-- {
		inter := s.unaryInterceptors[i]
		next := chained
		chained = func(c context.Context, fr *Frame) (*Frame, error) {
			return inter(c, fr, next)
		}
	}

	// -----------------------
	// Execute
	// -----------------------
	resp, err := chained(ctx, f)
	if err != nil {
		resp = &Frame{
			Type:     FrameTypeError,
			StreamID: f.StreamID,
			Method:   f.Method,
			Metadata: f.Metadata,
			Payload:  []byte(err.Error()),
		}
	}

	_ = WriteFrame(conn, resp)
}
