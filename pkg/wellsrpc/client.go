package wellsrpc

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type pendingResponse struct {
	ch chan *Frame
}

type RPCClient struct {
	conn    net.Conn
	pending map[uint32]*pendingResponse
	mu      sync.Mutex

	nextStream uint32

	unaryInterceptors []UnaryClientInterceptor

	streams   map[uint32]*Stream
	streamsMu sync.Mutex

	closed chan struct{}
}

func NewRPCClient(conn net.Conn) *RPCClient {
	c := &RPCClient{
		conn:    conn,
		pending: make(map[uint32]*pendingResponse),
		streams: make(map[uint32]*Stream),
		closed:  make(chan struct{}),
	}
	go c.readLoop()
	return c
}

func Dial(addr string, tlsCfg *tls.Config) (*RPCClient, error) {
	var conn net.Conn
	var err error
	if tlsCfg != nil {
		conn, err = tls.Dial("tcp", addr, tlsCfg)
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return nil, err
	}
	return NewRPCClient(conn), nil
}

func (c *RPCClient) Close() error {
	select {
	case <-c.closed:
		return nil
	default:
		close(c.closed)
		return c.conn.Close()
	}
}

func (c *RPCClient) nextStreamID() uint32 {
	v := atomic.AddUint32(&c.nextStream, 1)
	if v == 0 {
		v = atomic.AddUint32(&c.nextStream, 1)
	}
	return v
}

func (c *RPCClient) readLoop() {
	for {
		select {
		case <-c.closed:
			return
		default:
		}
		frame, err := ReadFrame(c.conn)
		if err != nil {
			c.mu.Lock()
			for _, p := range c.pending {
				select {
				case p.ch <- &Frame{Type: FrameTypeError, Payload: []byte(err.Error())}:
				default:
				}
			}
			c.mu.Unlock()
			return
		}
		c.mu.Lock()
		p := c.pending[frame.StreamID]
		c.mu.Unlock()
		if p != nil {
			select {
			case p.ch <- frame:
			default:
			}
			continue
		}
		c.streamsMu.Lock()
		st := c.streams[frame.StreamID]
		c.streamsMu.Unlock()
		if st != nil {
			switch frame.Type {
			case FrameTypeStreamData:
				select {
				case st.recvCh <- frame.Payload:
				default:
				}
			case FrameTypeStreamClose:
				st.Close()
				c.streamsMu.Lock()
				delete(c.streams, frame.StreamID)
				c.streamsMu.Unlock()
			}
		}
	}
}

func (c *RPCClient) UseUnaryInterceptor(i UnaryClientInterceptor) {
	c.unaryInterceptors = append(c.unaryInterceptors, i)
}

func (c *RPCClient) Call(ctx context.Context, method string, req WelliMarshaller, resp WelliMarshaller) error {
	reqData := req.MarshalWelli()
	invoke := func(ctx context.Context, payload []byte) ([]byte, error) {
		streamID := c.nextStreamID()
		p := &pendingResponse{ch: make(chan *Frame, 1)}
		c.mu.Lock()
		c.pending[streamID] = p
		c.mu.Unlock()
		defer func() {
			c.mu.Lock()
			delete(c.pending, streamID)
			c.mu.Unlock()
		}()

		f := &Frame{Type: FrameTypeRequest, StreamID: streamID, Method: method, Payload: payload}
		c.mu.Lock()
		err := WriteFrame(c.conn, f)
		c.mu.Unlock()
		if err != nil {
			return nil, err
		}
		select {
		case rf := <-p.ch:
			if rf == nil {
				return nil, errors.New("nil frame")
			}
			switch rf.Type {
			case FrameTypeResponse:
				return rf.Payload, nil
			case FrameTypeError:
				return nil, errors.New(string(rf.Payload))
			default:
				return nil, errors.New("unexpected frame type")
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	var chained func(ctx context.Context, payload []byte) ([]byte, error)
	chained = func(ctx context.Context, payload []byte) ([]byte, error) {
		return invoke(ctx, payload)
	}
	for i := len(c.unaryInterceptors) - 1; i >= 0; i-- {
		inter := c.unaryInterceptors[i]
		next := chained
		chained = func(ctx context.Context, payload []byte) ([]byte, error) {
			return inter(ctx, method, payload, next)
		}
	}

	ctx2 := ctx
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx2, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	out, err := chained(ctx2, reqData)
	if err != nil {
		return err
	}
	return resp.UnmarshalWelli(out)
}

func (c *RPCClient) OpenStream(ctx context.Context, method string) (*Stream, error) {
	streamID := c.nextStreamID()
	stream := newStream(streamID, func(data []byte) error {
		f := &Frame{Type: FrameTypeStreamData, StreamID: streamID, Payload: data}
		c.mu.Lock()
		defer c.mu.Unlock()
		return WriteFrame(c.conn, f)
	})
	c.streamsMu.Lock()
	c.streams[streamID] = stream
	c.streamsMu.Unlock()

	f := &Frame{Type: FrameTypeStreamOpen, StreamID: streamID, Method: method}
	c.mu.Lock()
	if err := WriteFrame(c.conn, f); err != nil {
		c.mu.Unlock()
		return nil, err
	}
	c.mu.Unlock()
	return stream, nil
}
