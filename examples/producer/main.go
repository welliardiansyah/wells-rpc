package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"time"

	wellsrpc "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc"
	codec "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc/codec_generated"
)

func main() {
	addr := "127.0.0.1:9000"

	// =========================
	// TLS (optional)
	// =========================
	var tlsCfg *tls.Config
	if _, err := os.Stat("examples/certs/client.crt"); err == nil {
		cert, _ := tls.LoadX509KeyPair(
			"examples/certs/client.crt",
			"examples/certs/client.key",
		)
		caCert, _ := os.ReadFile("examples/certs/ca.crt")
		caPool := x509.NewCertPool()
		caPool.AppendCertsFromPEM(caCert)

		tlsCfg = &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caPool,
		}
		log.Println("TLS enabled (client)")
	}

	// =========================
	// Dial
	// =========================
	client, err := wellsrpc.Dial(addr, tlsCfg)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer client.Close()

	// =========================
	// Unary Call (IDEMPOTENT)
	// =========================
	req := &codec.SensorReading{
		Timestamp:   time.Now().Unix(),
		Temperature: 25.3,
		Humidity:    60.5,
		Payload:     []byte("hello unary"),
	}

	var ack codec.Ack

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// idempotency-key wajib untuk retry-safe
	ctx = context.WithValue(ctx, "idempotency-key", "sensor-send-001")

	if err := client.Call(
		ctx,
		"SensorService.SendReading",
		req,
		&ack,
	); err != nil {

		// RPCError-aware handling
		if re, ok := err.(*wellsrpc.RPCError); ok {
			log.Printf(
				"RPC error code=%s retryable=%v msg=%s\n",
				re.Code,
				re.Retryable,
				re.Message,
			)
		}
		log.Fatal("rpc unary err:", err)
	}

	fmt.Println("Got unary ack:", ack.Success)

	// =========================
	// Stream Call
	// =========================
	streamCtx := context.Background()
	stream, err := client.OpenStream(
		streamCtx,
		"SensorService.StreamReadings",
	)
	if err != nil {
		log.Fatal("open stream:", err)
	}

	for i := 0; i < 3; i++ {
		msg := &codec.SensorReading{
			Timestamp:   time.Now().Unix(),
			Temperature: 20 + float32(i),
			Humidity:    50,
			Payload:     []byte(fmt.Sprintf("stream item %d", i)),
		}

		if err := stream.Send(msg.MarshalWells()); err != nil {
			log.Println("stream send err:", err)
			break
		}

		ctx2, cancel2 := context.WithTimeout(
			context.Background(),
			2*time.Second,
		)
		b, err := stream.Recv(ctx2)
		cancel2()

		if err != nil {
			log.Println("stream recv err:", err)
			break
		}

		var a codec.Ack
		if err := a.UnmarshalWells(b); err != nil {
			log.Println("ack decode err:", err)
			break
		}

		fmt.Println("stream ack:", a.Success)
		time.Sleep(100 * time.Millisecond)
	}

	stream.Close()
}
