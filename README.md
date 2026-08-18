# WWSrapport Go client

Official Go SDK for the WWSrapport API.

`DeriveBagReference`, `SearchRegistryByBag` and `GetReportVerification` expose the Solana attestation flow. `WebhookEvents` contains all 27 supported event types.

## Links

- API overview and Swagger: https://wwsrapport.nl/api/docs
- OpenAPI JSON: https://wwsrapport.nl/api/openapi.json
- Request API access: https://wwsrapport.nl/api/toegang-aanvragen
- GitHub organization: https://github.com/wwsrapport

## Install

```bash
go get github.com/wwsrapport/go-client/wwsrapport
```

## Use

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/wwsrapport/go-client/wwsrapport"
)

func main() {
	client := wwsrapport.NewClient(os.Getenv("WWSRAPPORT_API_KEY"), nil)

	report, err := client.CreateReport(context.Background(), map[string]any{
		"address": map[string]any{
			"postcode":     "3905RB",
			"house_number": "4",
			"country":      "NL",
		},
		"customer_reference": "crm-demo-001",
		"input": map[string]any{
			"living_area_m2": 53,
			"energy_label":   "E",
		},
	}, "crm-demo-001")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(report))
}
```

## Supported resources

- Property prefill
- Report validation, creation, listing, retrieval and recalculation
- Calculation JSON and improvement advice JSON
- WWS report and improvement advice PDF downloads
- Usage and rulesets
- Webhook endpoint management, test deliveries and retries
- OAuth 2.0 client credentials alongside API keys
- Public-sector request context (municipality, purpose, case and client reference)
- Batch jobs, human review, tenant exports and controlled offboarding

Use `NewOAuthClient` for client credentials and `WithRequestContext` to add the
approved public-sector context to requests. Existing `NewClient` API-key integrations
remain compatible.

## Webhooks

```go
ok := wwsrapport.VerifyWebhookSignature(
	[]byte(rawBody),
	r.Header.Get("WWS-Webhook-Timestamp"),
	r.Header.Get("WWS-Webhook-Signature"),
	os.Getenv("WWSRAPPORT_WEBHOOK_SECRET"),
)
```

## Development

```bash
go test ./...
```
