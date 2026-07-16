package cli

import (
	"net/http"
	"testing"

	"github.com/admiral-project/admiral/admiralctl/internal/client"
)

func newMockClient(t *testing.T, httpClient *http.Client) *client.Client {
	t.Helper()
	c, err := client.New("https://localhost", "fake-token", "", client.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("create mock client: %v", err)
	}
	return c
}
