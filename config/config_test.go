package config

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// everyKey is a config file setting every key Config declares to a distinct,
// non-zero value. Get is expected to land all of them, which is what pins the
// `mapstructure` tags to the keys the rest of profiler actually uses.
const everyKey = `
profilesFolder: /tmp/profiles
profilerFileName: .profiler-test
localProfiles:
  - alpha
preserveProfile: true
shell: /bin/dash
showRepoSourceInList: true
k8sSwitchNamespace: true
supportedRepositories:
  - local
  - vault
ssmProfiles:
  - ssm-alpha
ssmRegion: eu-west-1
ssmParameterTier: Advanced
ignoredFiles:
  - sample.env
consulProfiles:
  - consul-alpha
consulAddress: http://consul.example.com:8500
consulToken: consul-token
consulTokenFile: /tmp/consul-token
consulProfilesPath: profiler-test
vaultAddress: https://vault.example.com:8200
vaultToken: vault-token
vaultProfilesPath: secret/data/profiler-test
aws_access_key_id: AKIAEXAMPLE
aws_secret_access_key: aws-secret
aws_session_token: aws-session
`

// TestGetDecodesEveryConfigField is the guard for the drift this replaced:
// Get used to re-read all 20 keys by hand, so the `mapstructure` tags were
// decorative and half of them named keys nothing else used
// (consul_address for consulAddress, and a `mapstracture` typo on shell).
// Any field whose tag stops matching its config key now shows up here as a
// zero value.
func TestGetDecodesEveryConfigField(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()

	viper.SetConfigType("yml")
	if err := viper.ReadConfig(strings.NewReader(everyKey)); err != nil {
		t.Fatal(err)
	}

	cfg := Get()

	v := reflect.ValueOf(cfg)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)

		if v.Field(i).IsZero() {
			t.Errorf(
				"%s is zero: no value decoded from key %q",
				field.Name,
				field.Tag.Get("mapstructure"),
			)
		}
	}
}

// TestEveryKeyFixtureIsComplete keeps the fixture above honest: a field added
// to Config without a line in everyKey would otherwise be silently untested.
func TestEveryKeyFixtureIsComplete(t *testing.T) {
	var inFixture []string
	for _, line := range strings.Split(everyKey, "\n") {
		if key, _, found := strings.Cut(line, ":"); found &&
			!strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "-") {
			inFixture = append(inFixture, key)
		}
	}

	declared := configKeys()

	sort.Strings(inFixture)
	sort.Strings(declared)

	if !reflect.DeepEqual(inFixture, declared) {
		t.Errorf(
			"fixture keys and Config keys differ:\n fixture: %v\n Config:  %v",
			inFixture,
			declared,
		)
	}
}

// TestConfigKeysDoNotCollide guards the one way two tags can look distinct and
// still be the same setting: viper lowercases every key it stores.
func TestConfigKeysDoNotCollide(t *testing.T) {
	seen := map[string]string{}

	for _, key := range configKeys() {
		lower := strings.ToLower(key)

		if other, dup := seen[lower]; dup {
			t.Errorf("%q and %q are the same viper key", other, key)
		}

		seen[lower] = key
	}
}

// TestBindEnvsReachesEveryKey pins why bindEnvs exists: AutomaticEnv answers
// viper.Get for an unbound key but leaves it out of viper.AllKeys, so the
// struct Get decodes would miss it.
func TestBindEnvsReachesEveryKey(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()

	t.Setenv("PROFILER_VAULTTOKEN", "from-env")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("profiler")

	if got := Get().VaultToken; got != "" {
		t.Fatalf("VaultToken = %q before bindEnvs; the test no longer proves anything", got)
	}

	bindEnvs()

	if got := Get().VaultToken; got != "from-env" {
		t.Errorf("VaultToken = %q after bindEnvs, want %q", got, "from-env")
	}
}
