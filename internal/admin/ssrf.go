package admin

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// safeRemoteURL rejects non-http(s) and obvious SSRF targets (loopback/private/link-local/metadata).
func safeRemoteURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return fmt.Errorf("链接无效")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("仅允许 http/https")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("链接无效")
	}
	low := strings.ToLower(host)
	if low == "localhost" || strings.HasSuffix(low, ".localhost") || low == "metadata.google.internal" {
		return fmt.Errorf("禁止内网地址")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ipIsBad(ip) {
			return fmt.Errorf("禁止内网地址")
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("无法解析主机")
	}
	for _, ip := range ips {
		if ipIsBad(ip) {
			return fmt.Errorf("禁止内网地址")
		}
	}
	return nil
}

func ipIsBad(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

// ssrfHTTPClient dials only after re-checking the resolved IP (mitigates DNS rebinding).
func ssrfHTTPClient(timeout time.Duration) *http.Client {
	d := &net.Dialer{Timeout: 10 * time.Second}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: nil,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, err
				}
				var last error
				for _, ipa := range ips {
					if ipIsBad(ipa.IP) {
						last = fmt.Errorf("禁止内网地址")
						continue
					}
					c, err := d.DialContext(ctx, network, net.JoinHostPort(ipa.IP.String(), port))
					if err == nil {
						return c, nil
					}
					last = err
				}
				if last == nil {
					last = fmt.Errorf("连接失败")
				}
				return nil, last
			},
		},
	}
}
