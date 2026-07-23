package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	miyaconfig "miya-desktop/internal/config"
)

func TestFetchProviderModelsFromConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("unexpected authorization header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"model-z"},{"id":"model-a"}]}`))
	}))
	t.Cleanup(server.Close)

	app := NewApp(nil)
	models, err := app.FetchProviderModelsFromConfig("test", miyaconfig.ProviderConfig{
		Type:    "openai",
		APIBase: server.URL,
		APIKey:  "test-key",
	})
	if err != nil {
		t.Fatalf("FetchProviderModelsFromConfig() error = %v", err)
	}
	want := []string{"model-a", "model-z"}
	if !reflect.DeepEqual(models, want) {
		t.Fatalf("FetchProviderModelsFromConfig() = %v, want %v", models, want)
	}
}

func TestFetchProviderModelsFromConfigRequiresAPIKey(t *testing.T) {
	app := NewApp(nil)
	_, err := app.FetchProviderModelsFromConfig("test", miyaconfig.ProviderConfig{})
	if err == nil {
		t.Fatal("FetchProviderModelsFromConfig() error = nil, want API key error")
	}
}
