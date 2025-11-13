package wellsrpc

type WelliMarshaller interface {
	MarshalWells() []byte
	UnmarshalWells([]byte) error
}

func Marshal(msg WelliMarshaller) []byte {
	return msg.MarshalWells()
}

func Unmarshal(msg WelliMarshaller, b []byte) error {
	return msg.UnmarshalWells(b)
}
