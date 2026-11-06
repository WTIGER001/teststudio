package buildinfo

import (
	"encoding/base64"
	"testing"
)

func TestManifestDecodingAndUnversionedFallback(t *testing.T) {
	input := base64.StdEncoding.EncodeToString([]byte(`{"version":"0.1.0-dev","displayVersion":"0.1.0.dev","buildId":"0123456789ab","builtAt":"2026-11-06T14:00:00Z","commit":"abc123","modified":true}`))
	value := decode(input)
	if value.DisplayVersion != "0.1.0.dev" || value.BuildID != "0123456789ab" || value.BuiltAt != "2026-11-06T14:00:00Z" || !value.Modified {
		t.Fatal(value)
	}
	for _, invalid := range []string{"", "invalid", base64.StdEncoding.EncodeToString([]byte(`{"version":"0.1.0-dev"}`))} {
		fallback := decode(invalid)
		if fallback.Version != "development" || fallback.BuiltAt != "" || fallback.BuildID != "" {
			t.Fatal("invalid manifest invents build metadata", fallback)
		}
	}
}
