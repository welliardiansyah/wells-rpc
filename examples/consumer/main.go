package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"

	wellsrpc "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc"
	codec "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc/codec_generated"
)

func main() {
	addr := "127.0.0.1:9000"
	srv := wellsrpc.NewRPCServer()

	if _, err := os.Stat("examples/certs/server.crt"); err == nil {
		cert, _ := tls.LoadX509KeyPair("examples/certs/server.crt", "examples/certs/server.key")
		caCert, _ := os.ReadFile("examples/certs/ca.crt")
		caPool := x509.NewCertPool()
		caPool.AppendCertsFromPEM(caCert)
		srv.WithTLS(&tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientCAs:    caPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
		})
		log.Println("TLS enabled")
	}

	srv.UseUnaryInterceptor(func(ctx context.Context, payload []byte, handler func(ctx context.Context, payload []byte) ([]byte, error)) ([]byte, error) {
		log.Printf("Received unary call with payload size: %d bytes\n", len(payload))
		return handler(ctx, payload)
	})

	srv.Register("SensorService.SendReading", func(ctx context.Context, payload []byte) ([]byte, error) {
		var req codec.SensorReading
		if err := req.UnmarshalWells(payload); err != nil {
			return nil, err
		}
		fmt.Printf("received unary: ts=%d temp=%.2f hum=%.2f payload=%s\n",
			req.Timestamp, req.Temperature, req.Humidity, string(req.Payload))
		ack := codec.Ack{Success: true}
		return ack.MarshalWells(), nil
	})

	srv.RegisterStream("SensorService.StreamReadings", func(ctx context.Context, s *wellsrpc.Stream) error {
		for {
			msg, err := s.Recv(ctx)
			if err != nil {
				return err
			}
			var r codec.SensorReading
			if err := r.UnmarshalWells(msg); err != nil {
				continue
			}
			fmt.Printf("stream recv: ts=%d temp=%.2f payload=%s\n", r.Timestamp, r.Temperature, string(r.Payload))
			ack := codec.Ack{Success: true}
			_ = s.Send(ack.MarshalWells())
		}
	})

	log.Println("listening", addr)
	if err := srv.Serve(addr); err != nil {
		log.Fatal(err)
	}
}
