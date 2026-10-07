package schema

type Bluetooth struct {
	Proxy   bool `json:"proxy"`
	Speaker bool `json:"speaker"`
}

const DefaultBluetoothProxy = false

func DefaultBluetooth() Bluetooth { return Bluetooth{Proxy: DefaultBluetoothProxy} }
