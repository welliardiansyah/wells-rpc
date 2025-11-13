package wellsrpc

import (
	"encoding/binary"
	"math"
)

func WriteFloat32LE(buf *[]byte, val float32) {
	u := math.Float32bits(val)
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], u)
	*buf = append(*buf, tmp[:]...)
}

func WriteFloat64LE(buf *[]byte, val float64) {
	u := math.Float64bits(val)
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], u)
	*buf = append(*buf, tmp[:]...)
}

func ReadFloat32LE(b []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b))
}

func ReadFloat64LE(b []byte) float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(b))
}
