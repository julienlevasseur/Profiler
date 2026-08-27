package ssm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/spf13/viper"
)

// parameter is one entry as GetParametersByPath returns it: a full hierarchy
// name, and a value.
func parameter(name, value string) *ssm.Parameter {
	return &ssm.Parameter{
		Name:  aws.String(name),
		Value: aws.String(value),
	}
}

// TestProfileNames covers the two things a listing of /profiler has to survive:
// a profile appears once per variable it holds, and SSM returns the profile
// folders themselves, which are not variables.
func TestProfileNames(t *testing.T) {
	params := []*ssm.Parameter{
		parameter("/profiler/demo/", ""),
		parameter("/profiler/demo/profile_name", "demo"),
		parameter("/profiler/demo/FOO", "bar"),
		parameter("/profiler/demo/BAZ", "qux"),
		parameter("/profiler/other/profile_name", "other"),
	}

	got := profileNames(params)
	want := []string{"demo", "other"}

	if len(got) != len(want) {
		t.Fatalf("profileNames() = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("profileNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestProfileNamesEmpty is the empty-store case: nothing listed rather than a
// slice holding one empty name.
func TestProfileNamesEmpty(t *testing.T) {
	if got := profileNames(nil); len(got) != 0 {
		t.Errorf("profileNames(nil) = %v, want no profiles", got)
	}
}

func TestParametersToProfile(t *testing.T) {
	params := []*ssm.Parameter{
		parameter("/profiler/demo/", ""),
		parameter("/profiler/demo/profile_name", "demo"),
		parameter("/profiler/demo/FOO", "bar"),
	}

	p := parametersToProfile("demo", params)

	if p.Name != "demo" {
		t.Errorf("Name = %q, want %q", p.Name, "demo")
	}

	// The folder parameter is not a variable, so it is not a KV:
	if len(p.KVs) != 2 {
		t.Fatalf("KVs = %v, want the two variables", p.KVs)
	}

	want := map[string]string{"profile_name": "demo", "FOO": "bar"}
	for _, kv := range p.KVs {
		if want[kv.Key] != kv.Value {
			t.Errorf("KV %q = %q, want %q", kv.Key, kv.Value, want[kv.Key])
		}
	}
}

func TestParameterPath(t *testing.T) {
	if got := profilePath("demo"); got != "/profiler/demo" {
		t.Errorf("profilePath(demo) = %q", got)
	}

	if got := parameterPath("demo", "FOO"); got != "/profiler/demo/FOO" {
		t.Errorf("parameterPath(demo, FOO) = %q", got)
	}
}

// isolateAWS points the SDK at empty credentials and config files and blocks
// the metadata endpoint, so a test that expects "no credentials" gets that
// answer from the SDK rather than from whatever the machine running the test
// happens to have configured.
func isolateAWS(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	for _, f := range []string{"credentials", "config"} {
		path := filepath.Join(dir, f)
		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_PROFILE", "")

	viper.Set("ssmRegion", "us-east-1")
	t.Cleanup(func() { viper.Set("ssmRegion", "us-east-1") })
}

// TestIsConfiguredWithCredentials is what makes SSM profiles show up in
// `profiler list`: credentials the AWS chain can resolve, and a region.
func TestIsConfiguredWithCredentials(t *testing.T) {
	isolateAWS(t)
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAEXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")

	if !IsConfigured() {
		t.Error("IsConfigured() = false with credentials in the environment")
	}
}

// TestIsConfiguredWithoutCredentials is the case that keeps `profiler list`
// usable on a machine with no AWS setup: cmd/list skips a repository that says
// it is not configured.
func TestIsConfiguredWithoutCredentials(t *testing.T) {
	isolateAWS(t)

	if IsConfigured() {
		t.Error("IsConfigured() = true with no resolvable credentials")
	}
}

// TestIsConfiguredWithoutRegion: SSM cannot be reached without one, and every
// call would fail with MissingRegion.
func TestIsConfiguredWithoutRegion(t *testing.T) {
	isolateAWS(t)
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAEXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	viper.Set("ssmRegion", "")

	if IsConfigured() {
		t.Error("IsConfigured() = true with no region")
	}
}

// TestAddProfileRejectsOddArguments: a key with no value never reaches AWS.
func TestAddProfileRejectsOddArguments(t *testing.T) {
	isolateAWS(t)

	if err := AddProfile([]string{"demo", "FOO"}); err == nil {
		t.Error("AddProfile(demo FOO) = nil, want an error naming the missing value")
	}

	if err := AddProfile(nil); err == nil {
		t.Error("AddProfile(nil) = nil, want an error naming the missing profile")
	}
}
