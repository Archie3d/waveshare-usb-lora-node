package types

type MacAddress [6]byte

func (m MacAddress) AsByteArray() []byte {
	return m[0:6]
}
