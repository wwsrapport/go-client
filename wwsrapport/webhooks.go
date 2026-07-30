package wwsrapport

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

func VerifyWebhookSignature(payload []byte, timestampHeader string, signatureHeader string, secret string) bool {
	return VerifyWebhookSignatureWithTolerance(payload, timestampHeader, signatureHeader, secret, 5*time.Minute, time.Now())
}

func VerifyWebhookSignatureWithTolerance(payload []byte, timestampHeader string, signatureHeader string, secret string, tolerance time.Duration, now time.Time) bool {
	if len(payload) == 0 || timestampHeader == "" || signatureHeader == "" || secret == "" {
		return false
	}

	timestamp, err := strconv.ParseInt(timestampHeader, 10, 64)
	if err != nil {
		return false
	}

	if diff := now.Sub(time.Unix(timestamp, 0)); diff > tolerance || diff < -tolerance {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestampHeader))
	mac.Write([]byte("."))
	mac.Write(payload)
	expected := []byte("v1=" + hex.EncodeToString(mac.Sum(nil)))

	for _, part := range strings.Split(signatureHeader, ",") {
		if hmac.Equal(expected, []byte(strings.TrimSpace(part))) {
			return true
		}
	}

	return false
}

