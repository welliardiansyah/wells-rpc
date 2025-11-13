package benchmark

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	codec "wells-rpc/pkg/wellsrpc/codec_generated"
)

func generateDummyData() *codec.SensorReading {
	return &codec.SensorReading{
		Timestamp:   time.Now().Unix(),
		Temperature: float32(rand.Float64()*40 - 10),
		Humidity:    float32(rand.Float64() * 100),
		Payload:     []byte(randomString(64)),
	}
}

func randomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func generateBatchData(n int) []*codec.SensorReading {
	data := make([]*codec.SensorReading, n)
	for i := 0; i < n; i++ {
		data[i] = generateDummyData()
	}
	return data
}

func BenchmarkWellsBin_Encode(b *testing.B) {
	s := generateDummyData()
	for i := 0; i < b.N; i++ {
		_ = s.MarshalWelli()
	}
}

func BenchmarkWellsBin_Decode(b *testing.B) {
	s := generateDummyData()
	data := s.MarshalWelli()
	out := &codec.SensorReading{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = out.UnmarshalWelli(data)
	}
}

func BenchmarkJSON_Encode(b *testing.B) {
	s := generateDummyData()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(s)
	}
}

func BenchmarkJSON_Decode(b *testing.B) {
	s := generateDummyData()
	data, _ := json.Marshal(s)
	out := &codec.SensorReading{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = json.Unmarshal(data, out)
	}
}

func BenchmarkWellsBin_BatchEncode(b *testing.B) {
	batch := generateBatchData(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range batch {
			_ = s.MarshalWelli()
		}
	}
}

func BenchmarkJSON_BatchEncode(b *testing.B) {
	batch := generateBatchData(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range batch {
			_, _ = json.Marshal(s)
		}
	}
}

func BenchmarkWellsBin_ParallelEncode(b *testing.B) {
	s := generateDummyData()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = s.MarshalWelli()
		}
	})
}

func BenchmarkWellsBin_ParallelDecode(b *testing.B) {
	s := generateDummyData()
	data := s.MarshalWelli()
	b.RunParallel(func(pb *testing.PB) {
		out := &codec.SensorReading{}
		for pb.Next() {
			_ = out.UnmarshalWelli(data)
		}
	})
}

func BenchmarkJSON_ParallelEncode(b *testing.B) {
	s := generateDummyData()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = json.Marshal(s)
		}
	})
}

func BenchmarkJSON_ParallelDecode(b *testing.B) {
	s := generateDummyData()
	data, _ := json.Marshal(s)
	b.RunParallel(func(pb *testing.PB) {
		out := &codec.SensorReading{}
		for pb.Next() {
			_ = json.Unmarshal(data, out)
		}
	})
}

func BenchmarkWellsBin_LargePayload(b *testing.B) {
	s := generateDummyData()
	s.Payload = make([]byte, 10*1024*1024)
	for i := range s.Payload {
		s.Payload[i] = byte(rand.Intn(256))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.MarshalWelli()
	}
}

func BenchmarkJSON_LargePayload(b *testing.B) {
	s := generateDummyData()
	s.Payload = make([]byte, 10*1024*1024)
	for i := range s.Payload {
		s.Payload[i] = byte(rand.Intn(256))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(s)
	}
}

func BenchmarkWellsBin_Stress(b *testing.B) {
	data := generateBatchData(100000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range data {
			buf := s.MarshalWelli()
			out := &codec.SensorReading{}
			_ = out.UnmarshalWelli(buf)
		}
	}
}

func BenchmarkJSON_Stress(b *testing.B) {
	data := generateBatchData(100000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range data {
			buf, _ := json.Marshal(s)
			out := &codec.SensorReading{}
			_ = json.Unmarshal(buf, out)
		}
	}
}
