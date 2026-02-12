// Package stripe provides Stripe webhook signature verification
// without depending on the full stripe-go SDK.
package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultTolerance is the default webhook timestamp tolerance (5 minutes).
	DefaultTolerance = 5 * time.Minute
	signingVersion   = "v1"
)

var (
	ErrInvalidHeader    = errors.New("stripe: invalid Stripe-Signature header")
	ErrNoValidSignature = errors.New("stripe: no valid signature found")
	ErrTimestampExpired = errors.New("stripe: timestamp outside tolerance")
)

// ConstructEvent verifies the Stripe webhook signature and returns the raw payload.
// It mirrors stripe.webhook.ConstructEvent from the official SDK.
func ConstructEvent(payload []byte, sigHeader, secret string) error {
	return ConstructEventWithTolerance(payload, sigHeader, secret, DefaultTolerance)
}

// ConstructEventWithTolerance verifies with a custom time tolerance.
func ConstructEventWithTolerance(payload []byte, sigHeader, secret string, tolerance time.Duration) error {
	header, err := parseSignatureHeader(sigHeader)
	if err != nil {
		return err
	}

	// Check timestamp tolerance
	if tolerance > 0 {
		diff := time.Since(time.Unix(header.timestamp, 0))
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			return ErrTimestampExpired
		}
	}

	// Compute expected signature
	expectedSig := computeSignature(header.timestamp, payload, secret)

	// Compare against all v1 signatures
	for _, sig := range header.signatures {
		if hmac.Equal([]byte(sig), []byte(expectedSig)) {
			return nil
		}
	}
	return ErrNoValidSignature
}

type signatureHeader struct {
	timestamp  int64
	signatures []string
}

func parseSignatureHeader(header string) (signatureHeader, error) {
	var sh signatureHeader
	if strings.TrimSpace(header) == "" {
		return sh, ErrInvalidHeader
	}

	pairs := strings.Split(header, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				return sh, ErrInvalidHeader
			}
			sh.timestamp = ts
		case signingVersion:
			sh.signatures = append(sh.signatures, kv[1])
		}
	}

	if sh.timestamp == 0 || len(sh.signatures) == 0 {
		return sh, ErrInvalidHeader
	}
	return sh, nil
}

// ComputeSignatureForTest is exported for use in handler tests.
func ComputeSignatureForTest(timestamp int64, payload []byte, secret string) string {
	return computeSignature(timestamp, payload, secret)
}

func computeSignature(timestamp int64, payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d", timestamp)
	mac.Write([]byte("."))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
