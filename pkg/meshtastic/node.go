package meshtastic

type Node struct {
	Id         uint32
	ShortName  string
	LongName   string
	MacAddress []byte
	HwModel    string
	PublicKey  []byte
}
