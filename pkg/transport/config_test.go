package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// clearEnv unsets every variable resolve consults, so a test starts from a
// known state regardless of the developer's shell.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		EnvAPIKey, EnvRegion, EnvDomain, EnvAPIURL,
		EnvSandboxURL, EnvDebug, EnvInsecureHTTP,
	} {
		t.Setenv(name, "")
	}
}

func TestResolveEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		env        map[string]string
		cfg        Config
		wantRegion string
		wantDomain string
		wantAPIURL string
	}{
		{
			name:       "defaults",
			cfg:        Config{APIKey: "k"},
			wantRegion: "cn-wlcb",
			wantDomain: "cn-wlcb.sandbox.ucloudai.com",
			wantAPIURL: "https://api.cn-wlcb.sandbox.ucloudai.com",
		},
		{
			name:       "region expands to domain",
			cfg:        Config{APIKey: "k", Region: "us-ca"},
			wantRegion: "us-ca",
			wantDomain: "us-ca.sandbox.ucloudai.com",
			wantAPIURL: "https://api.us-ca.sandbox.ucloudai.com",
		},
		{
			name:       "domain wins over region, and region is read back from it",
			cfg:        Config{APIKey: "k", Region: "us-ca", Domain: "gray-1.sandbox.ucloudai.com"},
			wantRegion: "gray-1",
			wantDomain: "gray-1.sandbox.ucloudai.com",
			wantAPIURL: "https://api.gray-1.sandbox.ucloudai.com",
		},
		{
			name:       "private domain keeps the configured region",
			cfg:        Config{APIKey: "k", Region: "us-ca", Domain: "sandbox.internal.corp"},
			wantRegion: "us-ca",
			wantDomain: "sandbox.internal.corp",
			wantAPIURL: "https://api.sandbox.internal.corp",
		},
		{
			name:       "private domain with no region falls back to the default",
			cfg:        Config{APIKey: "k", Domain: "sandbox.internal.corp"},
			wantRegion: "cn-wlcb",
			wantDomain: "sandbox.internal.corp",
			wantAPIURL: "https://api.sandbox.internal.corp",
		},
		{
			name:       "api url wins over domain",
			cfg:        Config{APIKey: "k", Domain: "us-ca.sandbox.ucloudai.com", APIURL: "http://10.0.0.5:8080"},
			wantRegion: "us-ca",
			wantDomain: "us-ca.sandbox.ucloudai.com",
			wantAPIURL: "http://10.0.0.5:8080",
		},
		{
			name:       "api url loses its trailing slash",
			cfg:        Config{APIKey: "k", APIURL: "http://10.0.0.5:8080/"},
			wantRegion: "cn-wlcb",
			wantDomain: "cn-wlcb.sandbox.ucloudai.com",
			wantAPIURL: "http://10.0.0.5:8080",
		},
		{
			name:       "insecure http switches the derived scheme",
			cfg:        Config{APIKey: "k", InsecureHTTP: true},
			wantRegion: "cn-wlcb",
			wantDomain: "cn-wlcb.sandbox.ucloudai.com",
			wantAPIURL: "http://api.cn-wlcb.sandbox.ucloudai.com",
		},
		{
			name:       "region env var",
			env:        map[string]string{EnvRegion: "us-ca"},
			cfg:        Config{APIKey: "k"},
			wantRegion: "us-ca",
			wantDomain: "us-ca.sandbox.ucloudai.com",
			wantAPIURL: "https://api.us-ca.sandbox.ucloudai.com",
		},
		{
			name:       "region env var wins over domain env var",
			env:        map[string]string{EnvRegion: "us-ca", EnvDomain: "gray-1.sandbox.ucloudai.com"},
			cfg:        Config{APIKey: "k"},
			wantRegion: "us-ca",
			wantDomain: "us-ca.sandbox.ucloudai.com",
			wantAPIURL: "https://api.us-ca.sandbox.ucloudai.com",
		},
		{
			name:       "domain env var",
			env:        map[string]string{EnvDomain: "gray-1.sandbox.ucloudai.com"},
			cfg:        Config{APIKey: "k"},
			wantRegion: "gray-1",
			wantDomain: "gray-1.sandbox.ucloudai.com",
			wantAPIURL: "https://api.gray-1.sandbox.ucloudai.com",
		},
		{
			name:       "explicit config wins over env vars",
			env:        map[string]string{EnvRegion: "us-ca", EnvAPIURL: "http://from-env"},
			cfg:        Config{APIKey: "k", Region: "gray-1", APIURL: "http://from-config"},
			wantRegion: "gray-1",
			wantDomain: "gray-1.sandbox.ucloudai.com",
			wantAPIURL: "http://from-config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := tt.cfg.resolve()
			require.NoError(t, err)
			assert.Equal(t, tt.wantRegion, got.region, "region")
			assert.Equal(t, tt.wantDomain, got.domain, "domain")
			assert.Equal(t, tt.wantAPIURL, got.apiURL, "apiURL")
		})
	}
}

func TestResolveAPIKey(t *testing.T) {
	t.Run("from config", func(t *testing.T) {
		clearEnv(t)
		got, err := Config{APIKey: "from-config"}.resolve()
		require.NoError(t, err)
		assert.Equal(t, "from-config", got.apiKey)
	})

	t.Run("from environment", func(t *testing.T) {
		clearEnv(t)
		t.Setenv(EnvAPIKey, "from-env")
		got, err := Config{}.resolve()
		require.NoError(t, err)
		assert.Equal(t, "from-env", got.apiKey)
	})

	t.Run("missing is an error, not a client that will 401", func(t *testing.T) {
		clearEnv(t)
		_, err := Config{}.resolve()
		require.Error(t, err)
		assert.ErrorIs(t, err, errdefs.ErrInvalidArgument)
	})
}

func TestResolveRetries(t *testing.T) {
	clearEnv(t)

	t.Run("nil means the default", func(t *testing.T) {
		got, err := Config{APIKey: "k"}.resolve()
		require.NoError(t, err)
		assert.Equal(t, DefaultRetries, got.retries)
	})

	t.Run("zero disables retrying", func(t *testing.T) {
		zero := 0
		got, err := Config{APIKey: "k", Retries: &zero}.resolve()
		require.NoError(t, err)
		assert.Zero(t, got.retries)
	})

	t.Run("negative is rejected", func(t *testing.T) {
		negative := -1
		_, err := Config{APIKey: "k", Retries: &negative}.resolve()
		assert.Error(t, err)
	})
}

func TestTLSVerificationIsOnByDefault(t *testing.T) {
	clearEnv(t)

	// Earlier versions of this SDK disabled certificate verification
	// unconditionally. It is now opt-in, and this test is what keeps it that
	// way.
	got, err := Config{APIKey: "k"}.resolve()
	require.NoError(t, err)
	assert.False(t, got.insecureSkipTLS,
		"TLS verification must be on unless Config.InsecureSkipTLS asks otherwise")
}

func TestIsRegionCN(t *testing.T) {
	tests := map[string]bool{
		"cn-wlcb": true,
		"cn":      true,
		"CN-WLCB": true,
		" cn-sh ": true,
		"gray-1":  true,
		"us-ca":   false,
		"eu-fra":  false,
		"":        false,
	}
	for region, want := range tests {
		assert.Equal(t, want, IsRegionCN(region), "IsRegionCN(%q)", region)
	}
}

func TestRegionFromDomain(t *testing.T) {
	tests := map[string]string{
		"cn-wlcb.sandbox.ucloudai.com": "cn-wlcb",
		"gray-1.sandbox.ucloudai.com":  "gray-1",
		"sandbox.ucloudai.com":         "",
		"a.b.sandbox.ucloudai.com":     "",
		"sandbox.internal.corp":        "",
		"":                             "",
	}
	for domain, want := range tests {
		assert.Equal(t, want, regionFromDomain(domain), "regionFromDomain(%q)", domain)
	}
}
