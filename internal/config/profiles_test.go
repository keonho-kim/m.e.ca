// Purpose: Verify provider-scoped model aliases persist and cannot conflict or escape profile limits.
// Pattern: Configuration boundary tests.
// Usage: go test ./internal/config -run TestModelProfile.
// Related: internal/config/profiles.go, internal/provider/profile/profile.go.
package config

import (
	"reflect"
	"testing"

	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
)

func TestModelProfileMappingsRoundTripAndStayConnectionScoped(t *testing.T) {
	c := Defaults()
	c.ModelProfiles = []ModelProfile{{Provider: modelconn.VLLM, Endpoint: "http://localhost:8000/v1", Model: "local-coder", Profile: "ornith-1.5-35b-a3b"}}
	dir := t.TempDir()
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil || !reflect.DeepEqual(loaded.ModelProfiles, c.ModelProfiles) {
		t.Fatalf("mappings=%+v %v", loaded.ModelProfiles, err)
	}
	connection, err := modelconn.New(modelconn.VLLM, "http://localhost:8000/v1")
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.ModelPreference(connection, "local-coder").Profile; got != "ornith-1.5-35b-a3b" {
		t.Fatalf("profile=%q", got)
	}
	if got := loaded.ModelPreference(connection, "other").Profile; got != "" {
		t.Fatalf("model scope leak: %s", got)
	}
	connection.Kind = modelconn.LiteLLM
	if got := loaded.ModelPreference(connection, "local-coder").Profile; got != "" {
		t.Fatalf("provider scope leak: %s", got)
	}
	other, err := modelconn.New(modelconn.VLLM, "http://localhost:8001/v1")
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.ModelPreference(other, "local-coder").Profile; got != "" {
		t.Fatalf("endpoint scope leak: %s", got)
	}
}

func TestModelProfileMappingsRejectInvalidOrAmbiguousBindings(t *testing.T) {
	valid := ModelProfile{Provider: modelconn.VLLM, Endpoint: "http://localhost:8000/v1", Model: "local-coder", Profile: "qwen3.6-27b"}
	for _, test := range []struct {
		name     string
		bindings []ModelProfile
	}{
		{"unknown profile", []ModelProfile{{Provider: valid.Provider, Endpoint: valid.Endpoint, Model: valid.Model, Profile: "qwen"}}},
		{"outside target", []ModelProfile{{Provider: valid.Provider, Endpoint: valid.Endpoint, Model: valid.Model, Profile: "glm-5.3-flash"}}},
		{"missing endpoint", []ModelProfile{{Provider: valid.Provider, Model: valid.Model, Profile: valid.Profile}}},
		{"model conflict", []ModelProfile{{Provider: valid.Provider, Endpoint: valid.Endpoint, Model: "google/gemma-4-31B-it", Profile: valid.Profile}}},
		{"duplicate endpoint", []ModelProfile{valid, {Provider: valid.Provider, Endpoint: "http://localhost:8000", Model: valid.Model, Profile: valid.Profile}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := Defaults()
			c.ModelProfiles = test.bindings
			if err := Save(t.TempDir(), c); err == nil {
				t.Fatal("invalid mapping accepted")
			}
		})
	}
}

func TestModelProfileEffortDefaultsValidatePerModel(t *testing.T) {
	binding := ModelProfile{Provider: modelconn.VLLM, Endpoint: "http://localhost:8000/v1", Model: "team-coder", Profile: "qwen3.8-27b", ReasoningEffort: "medium"}
	c := Defaults()
	c.ModelProfiles = []ModelProfile{binding}
	directory := t.TempDir()
	if err := Save(directory, c); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	connection, err := modelconn.New(binding.Provider, binding.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.ModelPreference(connection, binding.Model); got.ReasoningEffort != "medium" {
		t.Fatalf("effort lost: %+v", got)
	}
	c.ModelProfiles[0].ReasoningEffort = "high"
	if err := Save(directory, c); err == nil {
		t.Fatal("Qwen effort outside the declared set accepted")
	}
	c.ModelProfiles[0].Profile = "ornith-1.5-9b"
	c.ModelProfiles[0].ReasoningEffort = "low"
	if err := Save(directory, c); err == nil {
		t.Fatal("base family effort copied into a derivative")
	}
}
