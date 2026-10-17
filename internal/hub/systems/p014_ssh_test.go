//go:build testing

package systems

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	gliderssh "github.com/gliderlabs/ssh"
	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/hub/expirymap"
	"github.com/henrygd/beszel/internal/testutil/telemetry"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestP014SSHPlatformIngestion(t *testing.T) {
	for _, platform := range []string{"linux", "windows", "macos", "ios"} {
		for _, version := range []string{"0.19.0", "0.21.0"} {
			for _, generic := range []bool{false, true} {
				name := platform + "/" + version + "/typed"
				if generic {
					name = platform + "/" + version + "/generic"
				}
				t.Run(name, func(t *testing.T) {
					sys, app := newTestSystemWithHub(t)
					sys.ctx = context.Background()
					sys.Status = up
					sys.manager.zfsFetchMap = expirymap.New[zfsFetchState](time.Hour)
					t.Cleanup(sys.manager.zfsFetchMap.StopCleaner)
					f, err := telemetry.Load(platform)
					require.NoError(t, err)
					_, clientKey, err := ed25519.GenerateKey(rand.Reader)
					require.NoError(t, err)
					clientSigner, err := ssh.NewSignerFromKey(clientKey)
					require.NoError(t, err)
					_, serverKey, err := ed25519.GenerateKey(rand.Reader)
					require.NoError(t, err)
					serverSigner, err := ssh.NewSignerFromKey(serverKey)
					require.NoError(t, err)
					agent := &gliderssh.Server{Version: "beszel_" + version, PublicKeyHandler: func(_ gliderssh.Context, key gliderssh.PublicKey) bool {
						return bytes.Equal(key.Marshal(), clientSigner.PublicKey().Marshal())
					}, Handler: func(s gliderssh.Session) {
						var request common.HubRequest[cbor.RawMessage]
						if err := cbor.NewDecoder(s).Decode(&request); err != nil {
							t.Error(err)
							return
						}
						response := common.AgentResponse{}
						if request.Action == common.GetData {
							if generic {
								response.Data, _ = cbor.Marshal(f.Data)
							} else {
								response.SystemData = &f.Data
							}
						} else if request.Action == common.SyncNetworkMonitors {
							response.Data, _ = cbor.Marshal(map[int]any{})
						} else {
							response.Error = "unsupported fixture action"
						}
						if err := cbor.NewEncoder(s).Encode(response); err != nil {
							t.Error(err)
						}
					}}
					agent.AddHostKey(serverSigner)
					listener, err := net.Listen("tcp", "127.0.0.1:0")
					require.NoError(t, err)
					done := make(chan struct{})
					go func() { defer close(done); _ = agent.Serve(listener) }()
					t.Cleanup(func() { _ = agent.Close(); <-done })
					sys.Host, sys.Port, err = net.SplitHostPort(listener.Addr().String())
					require.NoError(t, err)
					sys.manager.sshConfig = &ssh.ClientConfig{User: "p014", Auth: []ssh.AuthMethod{ssh.PublicKeys(clientSigner)}, Timeout: time.Second, HostKeyCallback: ssh.FixedHostKey(serverSigner.PublicKey())}
					t.Cleanup(sys.closeSSHConnection)
					require.NoError(t, sys.update())
					records, err := app.FindAllRecords("system_stats")
					require.NoError(t, err)
					require.Len(t, records, 1)
					var got system.Stats
					require.NoError(t, records[0].UnmarshalJSONField("stats", &got))
					require.Equal(t, f.Data.Stats, got)
					require.Equal(t, sys.Id, records[0].GetString("system"))
				})
			}
		}
	}
}
