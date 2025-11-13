package wellsrpc

func EncodeVarint(x uint64) []byte {
	var buf [10]byte
	i := 0
	for x >= 0x80 {
		buf[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	buf[i] = byte(x)
	return buf[:i+1]
}

func DecodeVarint(b []byte) (uint64, int) {
	var x uint64
	var s uint
	for i, c := range b {
		if c < 0x80 {
			return x | uint64(c)<<s, i + 1
		}
		x |= uint64(c&0x7F) << s
		s += 7
	}
	return 0, 0
}
