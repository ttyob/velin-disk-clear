package network

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClient creates a client with an optional user-configured HTTP proxy.
// An empty proxy uses the system/default direct transport behavior.
func HTTPClient(proxyURL string, timeout time.Duration, redirect func(*http.Request, []*http.Request) error) (*http.Client, error) {
	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		baseTransport = &http.Transport{}
	}
	transport := baseTransport.Clone()
	// The empty setting means direct connection, independent of shell-level
	// HTTP_PROXY variables that may be present on the host.
	transport.Proxy = nil
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil, errors.New("HTTP 代理地址无效，应为 http://host:port")
		}
		transport.Proxy = http.ProxyURL(parsed)
	}
	client := &http.Client{Transport: transport, Timeout: timeout}
	if redirect != nil {
		client.CheckRedirect = redirect
	}
	return client, nil
}
