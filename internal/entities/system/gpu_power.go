package system

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/fxamacker/cbor/v2"
)

// A nonzero reading is self-describing. Only explicit zero presence needs extra
// state; ordinary struct values and current Agent encodings retain their existing
// omission rules. Missing/null power is unavailable, never an observation of 0W.
func (g GPUData) HasPower() bool          { return g.Power != 0 || g.powerZeroPresent }
func (g GPUData) HasPackagePower() bool   { return g.PowerPkg != 0 || g.packagePowerZeroPresent }
func (g *GPUData) SetPower(value float64) { g.Power = value; g.powerZeroPresent = value == 0 }
func (g *GPUData) SetPackagePower(value float64) {
	g.PowerPkg = value
	g.packagePowerZeroPresent = value == 0
}

type gpuWire GPUData

func (g *GPUData) UnmarshalJSON(data []byte) error {
	var value gpuWire
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if len(fields) == 0 {
		return errors.New("missing GPU snapshot")
	}
	present := func(key string) bool {
		v := bytes.TrimSpace(fields[key])
		return len(v) > 0 && !bytes.Equal(v, []byte("null"))
	}
	value.powerZeroPresent = value.Power == 0 && present("p")
	value.packagePowerZeroPresent = value.PowerPkg == 0 && present("pp")
	*g = GPUData(value)
	return nil
}

func (g GPUData) MarshalJSON() ([]byte, error) {
	value := struct {
		gpuWire
		Power        *float64 `json:"p,omitempty"`
		PackagePower *float64 `json:"pp,omitempty"`
	}{gpuWire: gpuWire(g)}
	if g.HasPower() {
		value.Power = &g.Power
	}
	if g.HasPackagePower() {
		value.PackagePower = &g.PowerPkg
	}
	return json.Marshal(value)
}

func (g *GPUData) UnmarshalCBOR(data []byte) error {
	var value gpuWire
	if err := cbor.Unmarshal(data, &value); err != nil {
		return err
	}
	var fields map[uint64]cbor.RawMessage
	if err := cbor.Unmarshal(data, &fields); err != nil {
		return err
	}
	if len(fields) == 0 {
		return errors.New("missing GPU snapshot")
	}
	present := func(key uint64) bool { v := fields[key]; return len(v) > 0 && !bytes.Equal(v, []byte{0xf6}) }
	value.powerZeroPresent = value.Power == 0 && present(4)
	value.packagePowerZeroPresent = value.PowerPkg == 0 && present(6)
	*g = GPUData(value)
	return nil
}

func (g GPUData) MarshalCBOR() ([]byte, error) {
	data, err := cbor.Marshal(gpuWire(g))
	if err != nil || (!g.powerZeroPresent && !g.packagePowerZeroPresent) {
		return data, err
	}
	// Current Agent structs take the unchanged fast path above. A decoded legacy
	// zero is preserved if this snapshot is subsequently re-encoded as CBOR.
	var fields map[uint64]cbor.RawMessage
	if err = cbor.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	if g.powerZeroPresent {
		fields[4] = cbor.RawMessage{0}
	}
	if g.packagePowerZeroPresent {
		fields[6] = cbor.RawMessage{0}
	}
	return cbor.Marshal(fields)
}
