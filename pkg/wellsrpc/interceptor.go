package wellsrpc

import "context"

type UnaryServerInterceptor func(ctx context.Context, payload []byte, handler func(ctx context.Context, payload []byte) ([]byte, error)) ([]byte, error)

type UnaryClientInterceptor func(ctx context.Context, method string, payload []byte, invoke func(ctx context.Context, payload []byte) ([]byte, error)) ([]byte, error)
