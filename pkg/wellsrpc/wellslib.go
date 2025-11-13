package wellib

type WelliMarshaller interface {
	MarshalWelli() []byte
	UnmarshalWelli([]byte) error
}

func Marshal(msg WelliMarshaller) []byte {
	return msg.MarshalWelli()
}

func Unmarshal(msg WelliMarshaller, b []byte) error {
	return msg.UnmarshalWelli(b)
}
