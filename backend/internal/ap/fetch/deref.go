package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ActorDocument struct {
	ID        string `json:"id"`
	Inbox     string `json:"inbox"`
	PublicKey struct {
		ID           string `json:"id"`
		Owner        string `json:"owner"`
		PublicKeyPEM string `json:"publicKeyPem"`
	} `json:"publicKey"`
}

func DerefActor(ctx context.Context, actorURL string) (ActorDocument, error) {
	parsedURL, err := url.Parse(actorURL)
	if err != nil {
		return ActorDocument{}, fmt.Errorf("parse actor url: %w", err)
	}
	if !strings.EqualFold(parsedURL.Scheme, "https") {
		return ActorDocument{}, fmt.Errorf("only https actor dereference is allowed")
	}
	if parsedURL.Host == "" {
		return ActorDocument{}, fmt.Errorf("actor url host is required")
	}

	if err := validatePublicHost(ctx, parsedURL.Hostname()); err != nil {
		return ActorDocument{}, err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			if !strings.EqualFold(req.URL.Scheme, "https") {
				return fmt.Errorf("redirected to non-https url")
			}
			return validatePublicHost(ctx, req.URL.Hostname())
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, actorURL, nil)
	if err != nil {
		return ActorDocument{}, fmt.Errorf("build deref request: %w", err)
	}
	req.Header.Set("Accept", `application/activity+json, application/ld+json; profile="https://www.w3.org/ns/activitystreams", application/json`)

	resp, err := client.Do(req)
	if err != nil {
		return ActorDocument{}, fmt.Errorf("deref request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ActorDocument{}, fmt.Errorf("deref returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ActorDocument{}, fmt.Errorf("read deref body: %w", err)
	}

	var actor ActorDocument
	if err := json.Unmarshal(body, &actor); err != nil {
		return ActorDocument{}, fmt.Errorf("decode actor json: %w", err)
	}
	if strings.TrimSpace(actor.ID) == "" {
		return ActorDocument{}, fmt.Errorf("actor document missing id")
	}

	return actor, nil
}

func validatePublicHost(ctx context.Context, host string) error {
	if host == "" {
		return fmt.Errorf("host is required")
	}

	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return fmt.Errorf("host ip is not public")
		}
		return nil
	}

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve host: %w", err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("host has no ip addresses")
	}

	for _, addr := range addrs {
		if !isPublicIP(addr.IP) {
			return fmt.Errorf("host resolves to non-public ip")
		}
	}
	return nil
}

func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}

	if ipv4 := ip.To4(); ipv4 != nil {
		// Carrier-grade NAT space.
		if ipv4[0] == 100 && ipv4[1] >= 64 && ipv4[1] <= 127 {
			return false
		}
	}

	return true
}
