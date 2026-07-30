package wwsrapport

import (
	"testing"
	"time"
)

func TestVerifyWebhookSignature(t *testing.T) {
	payload := []byte(`{"type":"webhook.test"}`)
	timestamp := "1710000000"
	signature := "v1=50ebb068d786d8e4b90f5c8d2f4d49a988df87705e9fa45eb761216e8e78c6a7"

	if !VerifyWebhookSignatureWithTolerance(payload, timestamp, signature, "whsec_test", 5*time.Minute, time.Unix(1710000000, 0)) {
		t.Fatal("expected signature to be valid")
	}
}

func TestVerifyWebhookSignatureRejectsExpiredTimestamp(t *testing.T) {
	payload := []byte(`{"type":"webhook.test"}`)
	if VerifyWebhookSignatureWithTolerance(payload, "1710000000", "v1=bad", "whsec_test", 5*time.Minute, time.Unix(1710001000, 0)) {
		t.Fatal("expected signature to be rejected")
	}
}
