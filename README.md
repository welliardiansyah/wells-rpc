<h1 align="center">WellsRPC</h1>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.18+-00ADD8?logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
  <img src="https://img.shields.io/badge/build-passing-brightgreen" alt="Build Status">
  <img src="https://img.shields.io/badge/tests-100%25-blue" alt="Test Coverage">
</p>

<p align="center">
  <strong>WellsRPC</strong> is a lightweight high-performance RPC and binary serialization library for Go,  
  designed for microservices, IoT, Kafka messaging, and cross-language communication.
</p>

<p align="center">
  <img src="./asset/wellsrpc-overview.png" alt="WellsRPC Overview" width="650">
</p>

<hr>

<h2>📌 Table of Contents</h2>
<ul>
  <li><a href="#about-the-project">About The Project</a></li>
  <li><a href="#features">Features</a></li>
  <li><a href="#project-structure">Project Structure</a></li>
  <li><a href="#installation">Installation</a></li>
  <li><a href="#usage">Usage</a></li>
  <li><a href="#idl-and-code-generation">IDL & Code Generation</a></li>
  <li><a href="#example-producer-and-consumer">Example Producer & Consumer</a></li>
  <li><a href="#benchmark">Benchmark</a></li>
  <li><a href="#testing">Testing</a></li>
  <li><a href="#development-workflow">Development Workflow</a></li>
  <li><a href="#contributing">Contributing</a></li>
  <li><a href="#license">License</a></li>
</ul>

<h2 id="about-the-project">ℹ️ About The Project</h2>
<p>
  <strong>WellsRPC</strong> is a high-performance Go library providing:
</p>
<ul>
  <li>Binary serialization with <b>ultra-fast marshalling/unmarshalling</b></li>
  <li>Lightweight RPC framework with streaming and unary calls</li>
  <li>IDL-based code generation for Go structs and RPC stubs</li>
  <li>Support for IoT, Kafka, and microservice messaging</li>
</ul>
<p>It is optimized for <b>speed, low memory footprint, and minimal dependencies</b>.</p>

<h2 id="features">🚀 Features</h2>
<ul>
  <li>Marshal/Unmarshal structs to binary efficiently</li>
  <li>Buffer pool to reduce GC pressure</li>
  <li>IDL-driven code generation for Go RPC client/server</li>
  <li>Cross-language compatible (Go ↔ Java)</li>
  <li>Lightweight RPC client & server with streaming support</li>
  <li>TCP transport, with optional TLS</li>
  <li>Minimal dependencies</li>
</ul>

<h2 id="project-structure">🗂️ Project Structure</h2>
<pre>
wells-rpc/
├── pkg/wellsrpc/               # Core library
│   ├── client.go
│   ├── encode.go
│   ├── frame.go
│   ├── interceptor.go
│   ├── netmsg.go
│   ├── pool.go
│   ├── server.go
│   ├── stream.go
│   ├── varint.go
│   └── codec_generated/        # Generated structs & RPC stubs
│       └── sensor.go
├── cmd/welli-codegen/          # CLI for IDL -> Go code
│   └── main.go
├── examples/                   # Example producer & consumer
│   ├── producer/main.go
│   └── consumer/main.go
├── examples/sensor/
│   └── sensor.wb.idl           # IDL file
└── benchmark/
    └── wells_bench_test.go
</pre>

<h2 id="installation">🛠️ Installation</h2>

<h3>Prerequisites</h3>
<ul>
  <li>Go 1.18 or higher</li>
  <li>Git</li>
</ul>

<h3>Install WellsRPC</h3>
<pre><code>go get github.com/welliardiansyah/wells-rpc
</code></pre>

<h2 id="usage">⚡ Usage</h2>

<h3>Encode & Decode Struct</h3>
<pre><code>package main

import (
    "fmt"
    "time"
    "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc/codec_generated"
)

func main() {
    s := &codec_generated.SensorReading{
        Timestamp:   time.Now().Unix(),
        Temperature: 25.5,
        Humidity:    60,
        Payload:     []byte("payload-abc"),
    }

    // Encode
    data := s.MarshalWells()
    fmt.Println("Encoded data:", data)

    // Decode
    s2 := &codec_generated.SensorReading{}
    if err := s2.UnmarshalWells(data); err != nil {
        panic(err)
    }
    fmt.Println("Decoded struct:", s2)
}
</code></pre>

<h2 id="idl-and-code-generation">📝 IDL & Code Generation</h2>

<p>Define schema in <code>.wb.idl</code> file:</p>
<pre><code>message SensorReading {
  1: int64 timestamp;
  2: float32 temperature;
  3: float32 humidity;
  4: bytes payload;
}

message Ack {
  1: bool success;
}

service SensorService {
  rpc SendSensorData(SensorReading) returns (Ack);
}
</code></pre>

<p>Generate Go structs & RPC stubs:</p>
<pre><code>cd wells-rpc
go run cmd/welli-codegen/main.go examples/sensor/sensor.wb.idl
</code></pre>

<p>Generated files are placed in <code>pkg/wellsrpc/codec_generated/</code> and include:</p>
<ul>
  <li>Structs with <code>MarshalWells()</code> & <code>UnmarshalWells()</code></li>
  <li>RPC client & server stubs</li>
</ul>

<h2 id="example-producer-and-consumer">📦 Example Producer & Consumer</h2>

<h3>Producer Example</h3>
<pre><code>package main

import (
    "context"
    "fmt"
    "time"

    "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc"
    "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc/codec_generated"
)

func main() {
    client, err := wellsrpc.Dial("127.0.0.1:9000", nil)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    req := &codec_generated.SensorReading{
        Timestamp:   time.Now().Unix(),
        Temperature: 27,
        Humidity:    55,
        Payload:     []byte("test"),
    }

    resp := &codec_generated.Ack{}
    if err := client.Call(context.Background(), "SensorService.SendSensorData", req, resp); err != nil {
        panic(err)
    }

    fmt.Println("Server response:", resp.Success)
}
</code></pre>

<h3>Server Example</h3>
<pre><code>package main

import (
    "context"
    "fmt"

    "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc"
    "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc/codec_generated"
)

type sensorServer struct{}

func (s *sensorServer) SendSensorData(ctx context.Context, req *codec_generated.SensorReading) (*codec_generated.Ack, error) {
    fmt.Println("Received:", req)
    return &codec_generated.Ack{Success: true}, nil
}

func main() {
    srv := wellsrpc.NewRPCServer()
    codec_generated.RegisterSensorService(srv, &sensorServer{})
    fmt.Println("Server listening on :9000")
    if err := srv.Serve(":9000"); err != nil {
        panic(err)
    }
}
</code></pre>

<h2 id="benchmark">⚙️ Benchmark</h2>
<pre><code>cd benchmark
go test -bench=. -benchmem
</code></pre>

<p>Example output:</p>
<pre><code>
BenchmarkWellsRpc_Encode-8        15384214   75 ns/op    32 B/op   1 allocs/op
BenchmarkWellsRpc_Decode-8        9546022    110 ns/op   48 B/op   2 allocs/op
BenchmarkJSON_Encode-8            712413     1680 ns/op  480 B/op   6 allocs/op
BenchmarkJSON_Decode-8            545829     2060 ns/op  550 B/op   8 allocs/op
</code></pre>

<h2 id="testing">🧪 Testing</h2>
<pre><code>go test ./...
</code></pre>

<h2 id="development-workflow">🛠 Development Workflow</h2>
<ol>
  <li>Write schema in <code>.wb.idl</code></li>
  <li>Run codegen to generate Go structs & RPC stubs</li>
  <li>Implement producer/consumer using <code>MarshalWells()</code>/<code>UnmarshalWells()</code></li>
  <li>Write unit tests and benchmarks</li>
  <li>Publish updates via GitHub & Go modules</li>
</ol>

<h2 id="contributing">🤝 Contributing</h2>
<p>Contributions are welcome! Fork the repo, create a feature branch, and submit a pull request.</p>

<h2 id="license">📄 License</h2>
<p>MIT License - see <a href="https://github.com/welliardiansyah/wells-rpc/blob/main/LICENSE.md">LICENSE</a> for details.</p>
