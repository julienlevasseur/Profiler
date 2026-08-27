package ssm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// getParametersByPathRequest is the part of the SSM request this fake reads.
type getParametersByPathRequest struct {
	Path      string `json:"Path"`
	NextToken string `json:"NextToken"`
}

// parameter is one entry of the fake store, in the shape SSM returns.
type parameter struct {
	Name  string `json:"Name"`
	Type  string `json:"Type"`
	Value string `json:"Value"`
}

// fakeSSM stands up something that answers GetParametersByPath the way the
// Parameter Store does, so the repository can be exercised without an AWS
// account. It pages one parameter at a time: pagination is the part of the
// listing most likely to be got wrong, and least likely to be noticed, since
// a truncated answer looks like a smaller profile set rather than an error.
func fakeSSM(t *testing.T, store []parameter) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if target := r.Header.Get("X-Amz-Target"); target != "AmazonSSM.GetParametersByPath" {
			http.Error(w, "unexpected target "+target, http.StatusBadRequest)
			return
		}

		var req getParametersByPathRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Everything under the requested path, as Recursive asks for:
		var matching []parameter
		for _, p := range store {
			if len(p.Name) >= len(req.Path) && p.Name[:len(req.Path)] == req.Path {
				matching = append(matching, p)
			}
		}

		from := 0
		if req.NextToken != "" {
			if _, err := fmt.Sscanf(req.NextToken, "%d", &from); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		resp := map[string]interface{}{}

		if from < len(matching) {
			resp["Parameters"] = matching[from : from+1]
			if from+1 < len(matching) {
				resp["NextToken"] = fmt.Sprintf("%d", from+1)
			}
		} else {
			resp["Parameters"] = []parameter{}
		}

		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	return srv
}

// useFakeSSM points profiler at srv, with credentials the AWS chain resolves
// without reaching for the machine's own AWS setup.
func useFakeSSM(t *testing.T, endpoint string) {
	t.Helper()

	dir := t.TempDir()
	for _, f := range []string{"credentials", "config"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAEXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")

	viper.Set("ssmRegion", "us-east-1")
	viper.Set("ssmEndpoint", endpoint)
	t.Cleanup(func() { viper.Set("ssmEndpoint", "") })
}

// demoStore is one profile of two variables plus a second profile, which is
// what `profiler list` and `profiler ssm show` are asked about below.
var demoStore = []parameter{
	{Name: "/profiler/demo/profile_name", Type: "String", Value: "demo"},
	{Name: "/profiler/demo/FOO", Type: "String", Value: "bar"},
	{Name: "/profiler/other/profile_name", Type: "String", Value: "other"},
}

// TestList is what M1 is for: SSM profiles reaching `profiler list`. Each
// profile is named once, however many variables it holds, and across pages.
func TestList(t *testing.T) {
	srv := fakeSSM(t, demoStore)
	useFakeSSM(t, srv.URL)

	profiles, err := NewSSMRepository().List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	want := []string{"demo", "other"}
	if len(profiles) != len(want) {
		t.Fatalf("List() = %v, want %v", profiles, want)
	}

	for i := range want {
		if profiles[i] != want[i] {
			t.Errorf("List()[%d] = %q, want %q", i, profiles[i], want[i])
		}
	}
}

// TestGet is the `profiler ssm use` path: the profile comes back with its
// values, which is what SetEnvironment exports.
func TestGet(t *testing.T) {
	srv := fakeSSM(t, demoStore)
	useFakeSSM(t, srv.URL)

	p, err := NewSSMRepository().Get("demo")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if p.Name != "demo" {
		t.Errorf("Get().Name = %q, want %q", p.Name, "demo")
	}

	got := map[string]string{}
	for _, kv := range p.KVs {
		got[kv.Key] = kv.Value
	}

	want := map[string]string{"profile_name": "demo", "FOO": "bar"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("Get() %s = %q, want %q", k, got[k], v)
		}
	}
}

func TestShow(t *testing.T) {
	srv := fakeSSM(t, demoStore)
	useFakeSSM(t, srv.URL)

	vars, err := NewSSMRepository().Show("demo")
	if err != nil {
		t.Fatalf("Show() error: %v", err)
	}

	if len(vars) != 2 {
		t.Fatalf("Show() = %v, want the two variables of demo", vars)
	}
}

// TestIsConfigured: the repository has to say yes before cmd/list asks it for
// anything at all.
func TestIsConfigured(t *testing.T) {
	srv := fakeSSM(t, demoStore)
	useFakeSSM(t, srv.URL)

	if !NewSSMRepository().IsConfigured() {
		t.Error("IsConfigured() = false with credentials and a region")
	}
}
