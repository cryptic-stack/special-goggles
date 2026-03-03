package signatures

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var (
	ErrMissingSignature = errors.New("missing Signature header")
	ErrMissingKeyID     = errors.New("missing keyId in signature")
	ErrUnknownKey       = errors.New("signature key not found")
)

type ParsedSignature struct {
	KeyID     string
	Algorithm string
	Headers   []string
	Signature string
}

type PublicKeyResolver func(ctx context.Context, keyID string) (string, error)

func VerifyRequest(
	ctx context.Context,
	req *http.Request,
	body []byte,
	maxSkew time.Duration,
	resolvePublicKey PublicKeyResolver,
) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}
	if resolvePublicKey == nil {
		return fmt.Errorf("resolver is required")
	}

	parsed, err := ParseSignatureHeader(req.Header.Get("Signature"))
	if err != nil {
		return err
	}

	if parsed.Algorithm != "" && !strings.EqualFold(parsed.Algorithm, "rsa-sha256") {
		return fmt.Errorf("unsupported signature algorithm: %s", parsed.Algorithm)
	}

	if parsed.KeyID == "" {
		return ErrMissingKeyID
	}

	if maxSkew > 0 {
		if err := verifyDateHeader(req, maxSkew); err != nil {
			return err
		}
	}

	if includesHeader(parsed.Headers, "digest") {
		if err := verifyDigestHeader(req, body); err != nil {
			return err
		}
	}

	signingString, err := BuildSigningString(req, parsed.Headers)
	if err != nil {
		return err
	}

	publicKeyPEM, err := resolvePublicKey(ctx, parsed.KeyID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(publicKeyPEM) == "" {
		return ErrUnknownKey
	}

	publicKey, err := parseRSAPublicKey(publicKeyPEM)
	if err != nil {
		return err
	}

	sig, err := base64.StdEncoding.DecodeString(parsed.Signature)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	hash := sha256.Sum256([]byte(signingString))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], sig); err != nil {
		return fmt.Errorf("verify signature: %w", err)
	}

	return nil
}

func ParseSignatureHeader(header string) (ParsedSignature, error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return ParsedSignature{}, ErrMissingSignature
	}

	params := map[string]string{}
	parts := splitSignatureParts(header)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(kv[0]))
		val := strings.TrimSpace(kv[1])
		val = strings.Trim(val, "\"")
		params[key] = val
	}

	parsed := ParsedSignature{
		KeyID:     params["keyid"],
		Algorithm: params["algorithm"],
		Signature: params["signature"],
	}

	headersRaw := strings.TrimSpace(params["headers"])
	if headersRaw == "" {
		parsed.Headers = []string{"date"}
	} else {
		for _, h := range strings.Fields(headersRaw) {
			parsed.Headers = append(parsed.Headers, strings.ToLower(strings.TrimSpace(h)))
		}
	}

	if parsed.Signature == "" {
		return ParsedSignature{}, fmt.Errorf("missing signature value")
	}

	return parsed, nil
}

func BuildSigningString(req *http.Request, signedHeaders []string) (string, error) {
	if len(signedHeaders) == 0 {
		signedHeaders = []string{"date"}
	}

	lines := make([]string, 0, len(signedHeaders))
	for _, h := range signedHeaders {
		key := strings.ToLower(strings.TrimSpace(h))
		if key == "" {
			continue
		}

		var value string
		switch key {
		case "(request-target)":
			value = strings.ToLower(req.Method) + " " + req.URL.RequestURI()
		case "host":
			value = req.Host
			if value == "" {
				value = req.URL.Host
			}
		default:
			value = req.Header.Get(key)
		}

		if strings.TrimSpace(value) == "" {
			return "", fmt.Errorf("missing signed header: %s", key)
		}
		lines = append(lines, key+": "+value)
	}

	if len(lines) == 0 {
		return "", fmt.Errorf("no signed headers provided")
	}

	return strings.Join(lines, "\n"), nil
}

func verifyDateHeader(req *http.Request, maxSkew time.Duration) error {
	dateHeader := req.Header.Get("Date")
	if strings.TrimSpace(dateHeader) == "" {
		return fmt.Errorf("missing Date header")
	}

	dateValue, err := http.ParseTime(dateHeader)
	if err != nil {
		return fmt.Errorf("invalid Date header: %w", err)
	}

	now := time.Now().UTC()
	delta := now.Sub(dateValue.UTC())
	if delta < 0 {
		delta = -delta
	}
	if delta > maxSkew {
		return fmt.Errorf("Date header outside allowed skew")
	}

	return nil
}

func verifyDigestHeader(req *http.Request, body []byte) error {
	got := strings.TrimSpace(req.Header.Get("Digest"))
	if got == "" {
		return fmt.Errorf("missing Digest header")
	}

	want := BuildDigestHeader(body)
	if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
		return fmt.Errorf("Digest mismatch")
	}
	return nil
}

func parseRSAPublicKey(publicKeyPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("invalid public key pem")
	}

	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("public key is not RSA")
		}
		return rsaPub, nil
	}

	if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
		rsaPub, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("certificate public key is not RSA")
		}
		return rsaPub, nil
	}

	return nil, fmt.Errorf("parse public key failed")
}

func includesHeader(headers []string, name string) bool {
	for _, h := range headers {
		if strings.EqualFold(strings.TrimSpace(h), name) {
			return true
		}
	}
	return false
}

func splitSignatureParts(header string) []string {
	var parts []string
	var cur strings.Builder
	inQuotes := false

	for _, r := range header {
		switch r {
		case '"':
			inQuotes = !inQuotes
			cur.WriteRune(r)
		case ',':
			if inQuotes {
				cur.WriteRune(r)
				continue
			}
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}

	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}
