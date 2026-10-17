// Package telemetry provides independently authored synthetic P014 Agent snapshots.
// These fixtures describe protocol capabilities; they are not native OS captures.
package telemetry

import (
	"embed"
	"encoding/json"

	"github.com/henrygd/beszel/internal/entities/system"
)

//go:embed testdata/*.json
var Files embed.FS

type Fixture struct {
	Platform string              `json:"platform"`
	SystemID string              `json:"system_id"`
	OwnerID  string              `json:"owner_id"`
	Data     system.CombinedData `json:"data"`
}

func Load(platform string) (Fixture, error) {
	var fixture Fixture
	data, err := Files.ReadFile("testdata/" + platform + ".json")
	if err != nil {
		return fixture, err
	}
	err = json.Unmarshal(data, &fixture)
	return fixture, err
}
