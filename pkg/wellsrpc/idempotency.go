package wellsrpc

import (
	"context"
	"sync"
	"time"
)

// =======================
// Idempotency Store
// =======================
// In-memory default store.
// Bisa diganti Redis / DB TANPA ubah interceptor.
type IdempotencyStore interface {
	Get(key string) (*Frame, bool)
	Set(key string, frame *Frame, ttl time.Duration)
	MarkInProgress(key string, ttl time.Duration) bool
	Delete(key string)
}

// =======================
// In-Memory Implementation
// =======================
type idempotencyEntry struct {
	frame     *Frame
	expiresAt time.Time
}

type MemoryIdempotencyStore struct {
	mu    sync.Mutex
	items map[string]*idempotencyEntry
}

func NewMemoryIdempotencyStore() *MemoryIdempotencyStore {
	s := &MemoryIdempotencyStore{
		items: make(map[string]*idempotencyEntry),
	}
	go s.gcLoop()
	return s
}

func (s *MemoryIdempotencyStore) Get(key string) (*Frame, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	it, ok := s.items[key]
	if !ok || time.Now().After(it.expiresAt) {
		return nil, false
	}
	if it.frame == nil {
		return nil, false
	}
	return it.frame, true
}

func (s *MemoryIdempotencyStore) Set(key string, frame *Frame, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[key] = &idempotencyEntry{
		frame:     frame,
		expiresAt: time.Now().Add(ttl),
	}
}

func (s *MemoryIdempotencyStore) MarkInProgress(key string, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if it, ok := s.items[key]; ok && time.Now().Before(it.expiresAt) {
		return false
	}

	s.items[key] = &idempotencyEntry{
		frame:     nil,
		expiresAt: time.Now().Add(ttl),
	}
	return true
}

func (s *MemoryIdempotencyStore) Delete(key string) {
	s.mu.Lock()
	delete(s.items, key)
	s.mu.Unlock()
}

func (s *MemoryIdempotencyStore) gcLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for k, v := range s.items {
			if now.After(v.expiresAt) {
				delete(s.items, k)
			}
		}
		s.mu.Unlock()
	}
}

// =======================
// Idempotency Interceptor
// =======================
func IdempotencyInterceptor(
	store IdempotencyStore,
	ttl time.Duration,
) UnaryServerInterceptor {

	return func(
		ctx context.Context,
		f *Frame,
		next func(context.Context, *Frame) (*Frame, error),
	) (*Frame, error) {

		// idempotency hanya untuk unary request
		if f.Type != FrameTypeRequest {
			return next(ctx, f)
		}

		key := f.Metadata["idempotency-key"]
		if key == "" {
			return next(ctx, f)
		}

		// 1️⃣ Jika sudah ada hasil → langsung return
		if cached, ok := store.Get(key); ok {
			return cached, nil
		}

		// 2️⃣ Tandai in-progress (hindari double execute)
		if !store.MarkInProgress(key, ttl) {
			return ErrorToFrame(
				f.StreamID,
				f.Method,
				f.Metadata,
				NewRequestInProgressError(),
			), nil
		}

		// 3️⃣ Eksekusi handler
		resp, err := next(ctx, f)
		if err != nil {
			store.Delete(key)
			return nil, err
		}

		// 4️⃣ Simpan response FINAL
		store.Set(key, cloneFrame(resp), ttl)

		return resp, nil
	}
}

// =======================
// Utilities
// =======================
func cloneFrame(f *Frame) *Frame {
	if f == nil {
		return nil
	}

	meta := make(map[string]string, len(f.Metadata))
	for k, v := range f.Metadata {
		meta[k] = v
	}

	payload := make([]byte, len(f.Payload))
	copy(payload, f.Payload)

	return &Frame{
		Type:     f.Type,
		StreamID: f.StreamID,
		Method:   f.Method,
		Metadata: meta,
		Payload:  payload,
	}
}
