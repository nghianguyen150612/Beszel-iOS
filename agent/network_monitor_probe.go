package agent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/henrygd/beszel"
	"github.com/henrygd/beszel/internal/entities/monitor"
)

const networkMonitorUserAgent = "Beszel-Agent/" + beszel.Version + " (+https://beszel.dev)"

// monitorProbe performs one check. Errors are recorded as loss by the task runner.
// Implementations must honor cancellation and bound their execution time.
type monitorProbe func(context.Context, monitor.Config) (int64, error)

// networkMonitorProbe returns a probe that uses client for HTTP monitors, or
// insecureClient for HTTP monitors that skip TLS certificate verification.
func networkMonitorProbe(client, insecureClient *http.Client) monitorProbe {
	return func(ctx context.Context, config monitor.Config) (int64, error) {
		switch config.Protocol {
		case "icmp":
			return monitorICMP(ctx, config.Target)
		case "tcp":
			return monitorTCP(ctx, config.Target, config.Port)
		case "http":
			if config.SkipTLSVerify {
				return monitorHTTP(ctx, insecureClient, config.Target)
			}
			return monitorHTTP(ctx, client, config.Target)
		case "dns":
			return monitorDNS(ctx, config.Target, config.Server)
		default:
			return -1, fmt.Errorf("unknown monitor protocol: %s", config.Protocol)
		}
	}
}

// monitorTCP measures connection establishment time, including address fallback
// but excluding DNS resolution.
// Returns -1 and an error on failure.
func monitorTCP(ctx context.Context, target string, port uint16) (int64, error) {
	return monitorTCPWith(ctx, target, port, net.DefaultResolver.LookupHost, func(ctx context.Context, address string, timeout time.Duration) (net.Conn, error) {
		dialer := net.Dialer{Timeout: timeout}
		return dialer.DialContext(ctx, "tcp", address)
	})
}

// Injection at the resolver/dial boundary allows deadline fairness to be tested
// without depending on a host route silently dropping packets.
func monitorTCPWith(ctx context.Context, target string, port uint16, lookup func(context.Context, string) ([]string, error), dial func(context.Context, string, time.Duration) (net.Conn, error)) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if port == 0 {
		return -1, errors.New("invalid TCP monitor port: 0")
	}

	// Resolve DNS first, outside the timing window but within the probe deadline.
	ips, err := lookup(ctx, target)
	if err != nil {
		return -1, err
	}
	if len(ips) == 0 {
		return -1, errors.New("no addresses resolved for TCP monitor")
	}
	portString := fmt.Sprintf("%d", port)
	deadline, _ := ctx.Deadline()

	// Share the remaining probe budget across addresses so an unresponsive
	// first address cannot consume all the time available for alternatives.
	start := time.Now()
	for i, ip := range ips {
		if err := ctx.Err(); err != nil {
			return -1, err
		}
		timeout := time.Until(deadline) / time.Duration(len(ips)-i)
		var conn net.Conn
		conn, err = dial(ctx, net.JoinHostPort(ip, portString), timeout)
		if err != nil {
			continue
		}
		responseUs := time.Since(start).Microseconds()
		conn.Close()
		return responseUs, nil
	}
	return -1, err
}

// monitorDNS measures DNS resolution response time in microseconds. If server is
// non-empty, the lookup is sent to that DNS server (host or host:port, default
// port 53) instead of the system resolver. Returns -1 and an error on failure.
func monitorDNS(ctx context.Context, target, server string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := ctx.Err(); err != nil {
		return -1, err
	}
	resolver := net.DefaultResolver
	if server != "" {
		if _, err := dnsServerAddress(server); err != nil {
			return -1, err
		}
		resolver = dnsResolverForServer(server)
	}

	start := time.Now()
	ips, err := resolver.LookupHost(ctx, target)
	if err != nil {
		return -1, err
	}
	if err := ctx.Err(); err != nil {
		return -1, err
	}
	if len(ips) == 0 {
		return -1, errors.New("no addresses resolved for DNS monitor")
	}
	return time.Since(start).Microseconds(), nil
}

// dnsResolverForServer builds a resolver that sends lookups to the given DNS
// server address instead of the system resolver. server may be a bare host or
// host:port; when no port is given, the standard DNS port 53 is used.
func dnsResolverForServer(server string) *net.Resolver {
	address, configErr := dnsServerAddress(server)
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			if configErr != nil {
				return nil, configErr
			}
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, address)
		},
	}
}

// dnsServerAddress accepts a host, bare/bracketed IPv6 literal or host:port.
// Invalid explicit configuration fails closed; it never selects the system resolver.
func dnsServerAddress(server string) (string, error) {
	host, port, err := net.SplitHostPort(server)
	if err != nil {
		host, port = strings.TrimPrefix(strings.TrimSuffix(server, "]"), "["), "53"
		if strings.Contains(host, ":") && net.ParseIP(host) == nil {
			return "", fmt.Errorf("invalid DNS server: %q", server)
		}
	}
	n, err := strconv.Atoi(port)
	if host == "" || strings.TrimSpace(host) != host || err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("invalid DNS server: %q", server)
	}
	return net.JoinHostPort(host, port), nil
}

// monitorHTTP measures HTTP GET request response in microseconds. Returns -1 and an error on failure.
func monitorHTTP(ctx context.Context, client *http.Client, url string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, monitor.MaxProbeTimeout)
	defer cancel()
	if client == nil {
		client = http.DefaultClient
	}
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return -1, err
	}
	req.Header.Set("User-Agent", networkMonitorUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return -1, err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return -1, fmt.Errorf("HTTP error: %s", resp.Status)
	}
	return time.Since(start).Microseconds(), nil
}
