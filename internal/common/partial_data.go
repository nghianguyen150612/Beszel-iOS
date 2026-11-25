package common

import (
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

// PartialDataVersion1 is the battery-only partial measurement protocol.
const PartialDataVersion1 uint8 = 1
const PartialMetricBattery uint8 = 1

// PartialData is a distinct generic GetData payload. Key 100 identifies the
// extension; it must never be decoded as a legacy CombinedData snapshot.
// An absent battery measurement denotes unavailable, while Percent=0 is valid.
type PartialData struct {
	Version      *uint8               `cbor:"100,keyasint"`
	Measurements []PartialMeasurement `cbor:"101,keyasint"`
}

type PartialMeasurement struct {
	ID      *uint8 `cbor:"0,keyasint"`
	Percent *uint8 `cbor:"1,keyasint"`
	State   *uint8 `cbor:"2,keyasint"`
}

// DecodePartialData validates the entire bounded payload before returning it.
// Unknown battery state is not sufficient to represent a reliable legacy
// battery tuple, so it is transmitted as an absent measurement.
func DecodePartialData(data []byte, negotiated uint8) (*PartialData, error) {
	if negotiated != PartialDataVersion1 || len(data) > 1024 {
		return nil, fmt.Errorf("unsupported partial version or oversized payload")
	}
	mode, err := (cbor.DecOptions{DupMapKey: cbor.DupMapKeyEnforcedAPF, ExtraReturnErrors: cbor.ExtraDecErrorUnknownField, MaxArrayElements: 16, MaxMapPairs: 16}).DecMode()
	if err != nil {
		return nil, err
	}
	var sample PartialData
	if err := mode.Unmarshal(data, &sample); err != nil {
		return nil, err
	}
	if sample.Version == nil || *sample.Version != negotiated || sample.Measurements == nil || len(sample.Measurements) > 1 {
		return nil, fmt.Errorf("invalid partial envelope")
	}
	for _, measurement := range sample.Measurements {
		if measurement.ID == nil || *measurement.ID != PartialMetricBattery || measurement.Percent == nil || *measurement.Percent > 100 || measurement.State == nil || *measurement.State < 1 || *measurement.State > 5 {
			return nil, fmt.Errorf("invalid partial battery measurement")
		}
	}
	return &sample, nil
}
