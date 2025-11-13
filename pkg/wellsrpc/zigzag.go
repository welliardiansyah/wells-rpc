package wellsrpc

func ZigzagEncode(n int64) uint64 {
	return uint64((n << 1) ^ (n >> 63))
}

func ZigzagDecode(u uint64) int64 {
	return int64(u>>1) ^ -int64(u&1)
}
