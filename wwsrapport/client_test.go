package wwsrapport

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func jsonResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestOAuthContextAndPublicSectorResources(t *testing.T) {
	tokenCalls, apiCalls := 0, 0
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path == "/oauth/token" {
			tokenCalls++
			return jsonResponse(`{"access_token":"oauth-token","expires_in":300}`), nil
		}
		apiCalls++
		if got := request.Header.Get("Authorization"); got != "Bearer oauth-token" {
			t.Fatalf("authorization = %q", got)
		}
		if got := request.Header.Get("X-WWS-Municipality-Code"); got != "GM0345" {
			t.Fatalf("municipality = %q", got)
		}
		if got := request.Header.Get("X-WWS-Case-Reference"); got != "ZAAK-1" {
			t.Fatalf("case = %q", got)
		}
		return jsonResponse(`{"data":{"id":"example"}}`), nil
	})}

	client := NewOAuthClient(OAuthClientCredentials{ClientID: "municipality", ClientSecret: "secret", TokenURL: "https://auth.example/oauth/token", Scopes: []string{"batches:write"}}, &RequestContext{MunicipalityCode: "GM0345", PurposeCode: "huurprijs-toezicht", CaseReference: "ZAAK-1", ClientReference: "zaaksysteem"}, httpClient)
	client.BaseURL = "https://api.example/v1"
	ctx := context.Background()
	if _, err := client.CreateBatch(ctx, map[string]any{"type": "address_check"}, "batch-key-123456"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.RequestTenantExport(ctx, "export-key-123456"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ReviewReport(ctx, "rpt_example", map[string]string{"status": "approved"}, "review-key-123456"); err != nil {
		t.Fatal(err)
	}
	if tokenCalls != 1 {
		t.Fatalf("token calls = %d", tokenCalls)
	}
	if apiCalls != 3 {
		t.Fatalf("api calls = %d", apiCalls)
	}
}
