package system

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// decodeByteMetrics accepts today's numeric JSON tuples and the byte strings
// emitted by older JSON v2 encoders. Decode through wider integers so v2 cannot
// reinterpret the numeric array as base64. Publish only a completely valid value.
func decodeByteMetrics(data []byte) ([]uint8, error) {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '"' {
		var legacy []byte
		err := json.Unmarshal(data, &legacy)
		return legacy, err
	}
	var values []uint16
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	if values == nil {
		return nil, nil
	}
	result := make([]uint8, len(values))
	for i, v := range values {
		if v > 255 {
			return nil, fmt.Errorf("byte metric at index %d exceeds 255", i)
		}
		result[i] = uint8(v)
	}
	return result, nil
}

func (s *Uint8Slice) UnmarshalJSON(data []byte) error {
	values, err := decodeByteMetrics(data)
	if err == nil {
		*s = Uint8Slice(values)
	}
	return err
}

func (b *Battery) UnmarshalJSON(data []byte) error {
	values, err := decodeByteMetrics(data)
	if err != nil {
		return err
	}
	if values == nil {
		*b = Battery{}
		return nil
	}
	if len(values) != 2 {
		return fmt.Errorf("battery tuple requires 2 values, got %d", len(values))
	}
	*b = Battery{values[0], values[1]}
	return nil
}
