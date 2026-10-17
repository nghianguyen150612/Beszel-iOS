package hub

import (
	"strings"
	"testing"
)

func TestHasTooManyByteRanges(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   bool
	}{
		{name: "sixteen byte ranges allowed", values: []string{"bytes=" + strings.Repeat("0-0,", 15) + "0-0"}},
		{name: "seventeen byte ranges rejected", values: []string{"bytes=" + strings.Repeat("0-0,", 16) + "0-0"}, want: true},
		{name: "range values are counted together", values: []string{"bytes=0-0", "bytes=1-1," + strings.Repeat("2-2,", 15) + "3-3"}, want: true},
		{name: "other units are ignored", values: []string{"items=" + strings.Repeat("0-0,", 20) + "0-0"}},
		{name: "malformed header without unit is ignored", values: []string{strings.Repeat("0-0,", 20) + "0-0"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hasTooManyByteRanges(test.values); got != test.want {
				t.Fatalf("hasTooManyByteRanges(%q) = %v, want %v", test.values, got, test.want)
			}
		})
	}
}
