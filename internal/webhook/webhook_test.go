package webhook_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

//nolint:noctx //test
func TestWebhookServer(t *testing.T) {
	// Set up a test handler.
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Test OK"))
	})

	// Use httptest to create a test server.
	ts := httptest.NewServer(testHandler)
	defer ts.Close()

	// Here, instead of starting an actual server, you can test requests against ts.
	res, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "Test OK" {
		t.Errorf("expected %q, got %q", "Test OK", string(body))
	}
}
