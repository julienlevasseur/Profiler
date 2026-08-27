package config

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	homedir "github.com/mitchellh/go-homedir"
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
ssmEndpoint: http://ssm.example.com:4566
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

// writeCfg writes a config file at path, failing the test rather than
// returning an error.
func writeCfg(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// initCfgSandbox runs InitCfg against a throwaway HOME with no PROFILER_CFG
// set, so it takes the default resolution path -- the one a fresh install
// hits. It returns the sandbox home. viper's global state and go-homedir's
// cache are both process-wide, so both are reset around the call.
func initCfgSandbox(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PROFILER_CFG", "") // empty is treated as unset by InitCfg

	homedir.DisableCache = true
	t.Cleanup(func() { homedir.DisableCache = false })

	viper.Reset()
	t.Cleanup(viper.Reset)

	InitCfg()

	return home
}

// TestInitCfgDefaultsProfilesFolder is R1: master defaulted profilesFolder to
// $HOME/.profiles and created it, then the default was commented out and the
// folder-creation helper deleted, so a fresh install resolved the path to ""
// and wrote every profile to the filesystem root. This pins both halves back
// down -- the default, and that the folder exists after InitCfg.
func TestInitCfgDefaultsProfilesFolder(t *testing.T) {
	home := initCfgSandbox(t)

	want := filepath.Join(home, ".profiles")

	if got := Get().ProfilesFolder; got != want {
		t.Errorf("ProfilesFolder = %q, want %q", got, want)
	}

	if fi, err := os.Stat(want); err != nil || !fi.IsDir() {
		t.Errorf("profiles folder %q was not created (err=%v)", want, err)
	}
}

// TestInitCfgSSMDefaults is R5: ssmRegion and ssmParameterTier lost their
// defaults in the new config package while three call sites still read them,
// so `profiler ssm list` failed with MissingRegion. The defaults are back;
// this is the test that keeps them.
func TestInitCfgSSMDefaults(t *testing.T) {
	initCfgSandbox(t)

	cfg := Get()

	if cfg.SSMRegion != "us-east-1" {
		t.Errorf("SSMRegion = %q, want us-east-1", cfg.SSMRegion)
	}
	if cfg.SSMParameterTier != "Standard" {
		t.Errorf("SSMParameterTier = %q, want Standard", cfg.SSMParameterTier)
	}
}

// TestInitCfgProfilerCfgAcceptsFileAndDirectory is R3: the redesign changed
// PROFILER_CFG from a file path (SetConfigFile) to a directory (AddConfigPath),
// silently breaking every setup that pointed it at a file. The decision
// recorded in the code is to accept both, and this is the test that agrees with
// it -- a directory holding the config, and a file that *is* the config, must
// resolve the same profilesFolder.
func TestInitCfgProfilerCfgAcceptsFileAndDirectory(t *testing.T) {
	cases := []struct {
		name  string
		asDir bool
	}{
		{"directory form (redesign)", true},
		{"file form (pre-redesign, must still resolve)", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			homedir.DisableCache = true
			t.Cleanup(func() { homedir.DisableCache = false })

			base := t.TempDir()
			profiles := filepath.Join(base, "profiles")
			body := "profilesFolder: " + profiles + "\n"

			var cfgEnv string
			if tc.asDir {
				writeCfg(t, filepath.Join(base, ConfigFileName+".yml"), body)
				cfgEnv = base
			} else {
				cfgEnv = filepath.Join(base, "myprofiler.yml")
				writeCfg(t, cfgEnv, body)
			}
			t.Setenv("PROFILER_CFG", cfgEnv)

			viper.Reset()
			t.Cleanup(viper.Reset)

			InitCfg()

			if got := Get().ProfilesFolder; got != profiles {
				t.Errorf("ProfilesFolder = %q, want %q", got, profiles)
			}
		})
	}
}
