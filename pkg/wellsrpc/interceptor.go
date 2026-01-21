package wellsrpc

import "context"

// =======================
// SERVER SIDE
// =======================
//
// Interceptor server bekerja di level FRAME,
// bukan payload, agar bisa:
// - membaca metadata
// - enforce deadline
// - idempotency
// - tracing
type UnaryServerInterceptor func(
	ctx context.Context,
	frame *Frame,
	handler func(ctx context.Context, frame *Frame) (*Frame, error),
) (*Frame, error)

// =======================
// CLIENT SIDE
// =======================
//
// Interceptor client juga bekerja di level FRAME,
// agar bisa:
// - inject metadata (trace-id, idempotency-key)
// - retry logic
// - deadline propagation
type UnaryClientInterceptor func(
	ctx context.Context,
	method string,
	frame *Frame,
	invoke func(ctx context.Context, frame *Frame) (*Frame, error),
) (*Frame, error)
