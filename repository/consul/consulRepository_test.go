package consul

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
)

// fakeConsul stands up something that answers /v1/status/leader the way a real
// agent does, so IsConfigured can be exercised without a Consul on the box.
// requireToken mirrors an agent with ACLs enabled.
func fakeConsul(t *testing.T, requireToken string) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/status/leader" {
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
			return
		}

		if requireToken != "" && r.Header.Get("X-Consul-Token") != requireToken {
			http.Error(w, "Permission denied", http.StatusForbidden)
			return
		}

		w.Write([]byte(`"127.0.0.1:8300"`))
	}))
	t.Cleanup(srv.Close)

	return srv
}

// setConsulCfg installs a consul configuration for the duration of one test.
func setConsulCfg(t *testing.T, address, token, tokenFile string) {
	t.Helper()

	for k, v := range map[string]string{
		"consulAddress":   address,
		"consulToken":     token,
		"consulTokenFile": tokenFile,
	} {
		viper.Set(k, v)
		t.Cleanup(func() { viper.Set(k, "") })
	}
}

// TestIsConfiguredWithoutToken is the case the old expression got wrong in the
// quiet direction: addr != "" && token != "" || tokenFile != "" reported a
// perfectly good ACL-less Consul as unconfigured, so it never showed up in
// `profiler list`.
func TestIsConfiguredWithoutToken(t *testing.T) {
	srv := fakeConsul(t, "")
	setConsulCfg(t, srv.URL, "", "")

	if !NewConsulRepository().IsConfigured() {
		t.Error("IsConfigured() = false for a reachable Consul with no auth; dev mode needs no token")
	}
}

// TestIsConfiguredWithToken keeps the authenticated case working.
func TestIsConfiguredWithToken(t *testing.T) {
	srv := fakeConsul(t, "s3cret")
	setConsulCfg(t, srv.URL, "s3cret", "")

	if !NewConsulRepository().IsConfigured() {
		t.Error("IsConfigured() = false for a reachable Consul with a valid token")
	}
}

// TestIsConfiguredTokenFileOnly is the reported half of G3: && binds tighter
// than ||, so a token file alone satisfied the check and `list` went looking
// for a Consul at an address nobody had configured.
func TestIsConfiguredTokenFileOnly(t *testing.T) {
	srv := fakeConsul(t, "")
	closedAddr := srv.URL
	srv.Close()

	setConsulCfg(t, closedAddr, "", "/nonexistent/consul-token")

	if NewConsulRepository().IsConfigured() {
		t.Error("IsConfigured() = true with a token file and nothing listening; a credential alone is not a Consul")
	}
}

// TestIsConfiguredUnreachable is what keeps `profiler list` alive on a machine
// with no Consul, now that consulAddress carries a localhost default: cmd/list
// exits non-zero if a repository claims to be configured and then fails.
func TestIsConfiguredUnreachable(t *testing.T) {
	srv := fakeConsul(t, "")
	closedAddr := srv.URL
	srv.Close()

	setConsulCfg(t, closedAddr, "", "")

	if NewConsulRepository().IsConfigured() {
		t.Error("IsConfigured() = true with nothing listening at consulAddress")
	}
}
