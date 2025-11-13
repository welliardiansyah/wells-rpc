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

	var tlsCfg *tls.Config
	if _, err := os.Stat("examples/certs/client.crt"); err == nil {
		cert, _ := tls.LoadX509KeyPair("examples/certs/client.crt", "examples/certs/client.key")
		caCert, _ := os.ReadFile("examples/certs/ca.crt")
		caPool := x509.NewCertPool()
		caPool.AppendCertsFromPEM(caCert)
		tlsCfg = &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caPool,
		}
	}

	client, err := wellsrpc.Dial(addr, tlsCfg)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer client.Close()

	req := &codec.SensorReading{
		Timestamp:   time.Now().Unix(),
		Temperature: 25.3,
		Humidity:    60.5,
		Payload:     []byte("hello unary"),
	}
	var ack codec.Ack
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Call(ctx, "SensorService.SendReading", req, &ack); err != nil {
		log.Fatal("rpc unary err:", err)
	}
	fmt.Println("Got unary ack:", ack.Success)

	stream, err := client.OpenStream(context.Background(), "SensorService.StreamReadings")
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
			log.Println("send:", err)
			break
		}
		ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
		b, err := stream.Recv(ctx2)
		cancel2()
		if err != nil {
			log.Println("recv ack err:", err)
			break
		}
		var a codec.Ack
		_ = a.UnmarshalWells(b)
		fmt.Println("stream ack:", a.Success)
		time.Sleep(100 * time.Millisecond)
	}
	stream.Close()
}
