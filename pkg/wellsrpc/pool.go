package wellsrpc

import "sync"

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 4096)
		return &b
	},
}

func GetBuffer() *[]byte {
	return bufPool.Get().(*[]byte)
}

func PutBuffer(b *[]byte) {
	*b = (*b)[:0]
	bufPool.Put(b)
}
