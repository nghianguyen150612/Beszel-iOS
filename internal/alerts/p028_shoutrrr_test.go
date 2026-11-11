package alerts

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/nicholas-fedor/shoutrrr/pkg/types"
	"github.com/stretchr/testify/require"
)

// Exercise the actual upgraded service with the same pre/post-initialization
// client injection used by public notifications; no external delivery occurs.
func TestP028ShoutrrrDeliveryContract(t *testing.T) {
	for _, result := range []string{"ok", "http-error", "timeout"} {
		t.Run(result, func(t *testing.T) {
			var requests int
			client := &http.Client{Transport: notificationRoundTripper(func(r *http.Request) (*http.Response, error) {
				requests++
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "public.example", r.URL.Hostname())
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.Contains(t, string(body), "P028 synthetic alert")
				if result == "timeout" {
					return nil, context.DeadlineExceeded
				}
				status := http.StatusOK
				if result == "http-error" {
					status = http.StatusInternalServerError
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("synthetic")), Request: r}, nil
			})}
			service, err := newPublicNotificationService("generic+https://public.example/notify", types.SenderOptions{HTTPClient: client})
			require.NoError(t, err)
			err = service.Send("P028 synthetic alert", &types.Params{})
			require.Equal(t, 1, requests, "keep the injected HTTP client")
			if result == "ok" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			if result == "timeout" {
				require.True(t, errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "deadline exceeded"))
			}
		})
	}
}
