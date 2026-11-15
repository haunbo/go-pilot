// Package xnet contains network helpers for service discovery and health checks.
package xnet

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// GetIPs returns non-loopback IPv4 addresses of the host.
func GetIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	ips := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		if ipv4 := ipNet.IP.To4(); ipv4 != nil {
			ips = append(ips, ipv4.String())
		}
	}
	return ips, nil
}

// GetLocalIP returns the first non-loopback IPv4 address.
func GetLocalIP() (string, error) {
	ips, err := GetIPs()
	if err != nil {
		return "", err
	}
	if len(ips) == 0 {
		return "", errors.New("xnet: no non-loopback IPv4 address")
	}
	return ips[0], nil
}

// GetMacAddrs returns non-empty hardware addresses.
func GetMacAddrs() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	addrs := make([]string, 0, len(interfaces))
	for _, iface := range interfaces {
		if mac := iface.HardwareAddr.String(); mac != "" {
			addrs = append(addrs, mac)
		}
	}
	return addrs, nil
}

// LocalListener listens on a random localhost port.
func LocalListener() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

// Dial checks TCP connectivity within timeout.
func Dial(addr string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = time.Second
	}
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	return conn.Close()
}

// HTTPPing calls url and requires a 2xx response.
func HTTPPing(ctx context.Context, url string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("xnet: unexpected status %d", resp.StatusCode)
	}
	return nil
}
