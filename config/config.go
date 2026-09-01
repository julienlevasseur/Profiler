package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

// Config is the whole of profiler's configuration. The `mapstructure` tag on
// each field is the key it is read from -- in the config file, and, uppercased
// and prefixed with PROFILER_, in the environment. Nothing copies these keys
// by hand: Get decodes the struct straight out of viper, and bindEnvs walks
// the same tags to register every key. A field is configurable because it is
// here, and by exactly the name written here.
type Config struct {
	ProfilesFolder        string   `mapstructure:"profilesFolder"`
	ProfilerFileName      string   `mapstructure:"profilerFileName"`
	LocalProfiles         []string `mapstructure:"localProfiles"`
	PreserveProfile       bool     `mapstructure:"preserveProfile"`
	Shell                 string   `mapstructure:"shell"`
	ShowRepoSourceInList  bool     `mapstructure:"showRepoSourceInList"`
	K8sSwitchNamespace    bool     `mapstructure:"k8sSwitchNamespace"`
	SupportedRepositories []string `mapstructure:"supportedRepositories"`
	SSMProfiles           []string `mapstructure:"ssmProfiles"`
	SSMRegion             string   `mapstructure:"ssmRegion"`
	SSMParameterTier      string   `mapstructure:"ssmParameterTier"`
	SSMEndpoint           string   `mapstructure:"ssmEndpoint"`
	ConsulProfiles        []string `mapstructure:"consulProfiles"`
	ConsulAddress         string   `mapstructure:"consulAddress"`
	ConsulToken           string   `mapstructure:"consulToken"`
	ConsulTokenFile       string   `mapstructure:"consulTokenFile"`
	ConsulProfilesPath    string   `mapstructure:"consulProfilesPath"`
	VaultAddress          string   `mapstructure:"vaultAddress"`
	VaultToken            string   `mapstructure:"vaultToken"`
	VaultProfilesPath     string   `mapstructure:"vaultProfilesPath"`
	IgnoredFiles          []string `mapstructure:"ignoredFiles"`
	// The AWS credentials keep their snake_case names: they mirror the
	// AWS_* environment variables, so PROFILER_AWS_ACCESS_KEY_ID reads the
	// way the unprefixed variable does.
	AWS_ACCESS_KEY_ID     string `mapstructure:"aws_access_key_id"`
	AWS_SECRET_ACCESS_KEY string `mapstructure:"aws_secret_access_key"`
	AWS_SESSION_TOKEN     string `mapstructure:"aws_session_token"`
}

// configKeys returns the viper key of every Config field, taken from its
// `mapstructure` tag.
func configKeys() []string {
	t := reflect.TypeOf(Config{})
	keys := make([]string, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		if key := t.Field(i).Tag.Get("mapstructure"); key != "" {
			keys = append(keys, key)
		}
	}

	return keys
}

// bindEnvs registers every Config key with viper, so PROFILER_<KEY> is picked
// up for all of them. AutomaticEnv alone would answer viper.Get for a key, but
// leave it out of viper.AllKeys -- and so out of the struct Get decodes.
func bindEnvs() {
	for _, key := range configKeys() {
		if err := viper.BindEnv(key); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

// ProfileAllowList returns the profile names repo is limited to, or nil when
// it is not limited. localProfiles, consulProfiles and ssmProfiles exist so a
// backend holding more profiles than this machine cares about -- a Consul
// shared between teams, an AWS account shared between stacks -- can be
// narrowed to the interesting ones. An unset or empty list means no limit, so
// leaving them out lists everything, as it always has.
//
// Vault has no equivalent option, and so is never narrowed.
func (c Config) ProfileAllowList(repo string) []string {
	switch repo {
	case "local":
		return c.LocalProfiles
	case "consul":
		return c.ConsulProfiles
	case "ssm":
		return c.SSMProfiles
	}

	return nil
}

// ConfigFileName is the config file's base name, without extension, as viper
// expects it.
const ConfigFileName = ".profiler_cfg"

// seedDefaultConfigFile writes a minimal config file at path if none exists
// yet, so a first run leaves something to discover and edit. A failure is
// reported but not fatal: the defaults set in InitCfg cover every setting, so
// profiler works without the file.
func seedDefaultConfigFile(path, home string) {
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		return
	}

	content := fmt.Sprintf(
		"# profiler configuration. See https://github.com/julienlevasseur/profiler#the-config-file\nprofilesFolder: %s\n",
		filepath.Join(home, ".profiles"),
	)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func InitCfg() {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	viper.AutomaticEnv()
	viper.SetConfigType("yml")

	// PROFILER_CFG accepts either form: an existing directory is searched for
	// the config file, anything else is the config file itself -- which is
	// what it meant before the redesign. A path that does not exist yet is
	// therefore a file, and gets seeded like the default one.
	if cfgPath := os.Getenv("PROFILER_CFG"); cfgPath != "" {
		if fi, err := os.Stat(cfgPath); err == nil && fi.IsDir() {
			viper.AddConfigPath(cfgPath)
			viper.SetConfigName(ConfigFileName)
		} else {
			seedDefaultConfigFile(cfgPath, home)
			viper.SetConfigFile(cfgPath)
		}
	} else {
		viper.AddConfigPath(home)
		viper.SetConfigName(ConfigFileName)
		seedDefaultConfigFile(filepath.Join(home, ConfigFileName+".yml"), home)
	}

	viper.SetEnvPrefix("profiler") // will be uppercased automatically

	viper.SetDefault("profilesFolder", filepath.Join(home, ".profiles"))

	viper.SetDefault("shell", os.Getenv("SHELL"))
	viper.SetDefault("preserveProfile", true)
	viper.SetDefault("profilerFileName", ".profiler")
	viper.SetDefault("showRepoSourceInList", false)
	viper.SetDefault("supportedRepositories", []string{
		//"azure",
		"local",
		"consul",
		"ssm",
		"vault",
	})

	bindEnvs()

	// Consul needs no auth in dev mode, so consulToken and consulTokenFile
	// stay optional. The address gets a localhost default, as vaultAddress
	// does below.
	viper.SetDefault("consulAddress", "http://127.0.0.1:8500")
	viper.SetDefault("consulProfilesPath", "profiler")

	viper.SetDefault("ssmRegion", "us-east-1")
	viper.SetDefault("ssmParameterTier", "Standard")
	// ssmEndpoint has no default on purpose: empty means the AWS endpoint
	// for ssmRegion, which is what all but the localstack/VPC-endpoint
	// cases want.

	viper.SetDefault("k8sSwitchNamespace", true)

	viper.SetDefault("vaultAddress", "http://127.0.0.1:8200")
	viper.SetDefault("vaultProfilesPath", "secret/metadata/profiler")

	viper.SetDefault(
		"ignoredFiles",
		[]string{
			"example.env",
			"sample.env",
		},
	)

	// A missing config file is not an error: every setting has a default.
	// Anything else (unreadable, malformed) is worth reporting -- on stderr,
	// so it never lands in the middle of `profiler list` output.
	var notFound viper.ConfigFileNotFoundError
	if err = viper.ReadInConfig(); err != nil && !errors.As(err, &notFound) {
		fmt.Fprintln(os.Stderr, err)
	}

	// Done last, so a profilesFolder set in the config file is the one
	// created rather than the default.
	if err := createProfilesFolder(viper.GetString("profilesFolder")); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

// createProfilesFolder creates the profiles folder if it does not exist yet.
func createProfilesFolder(path string) error {
	if path == "" {
		return errors.New("profilesFolder resolves to an empty path")
	}

	return os.MkdirAll(path, 0755)
}

// getMu serializes the viper reads Get makes. viper holds its configuration
// in plain maps with no locking of its own, and `profiler list` asks every
// repository for its profiles at once -- each of which calls Get -- so
// without this the concurrent decodes would race.
var getMu sync.Mutex

// Get returns the current configuration. It decodes viper on every call
// rather than caching, so a value set after InitCfg -- as the tests do with
// viper.Set -- is seen.
//
// It is safe to call from several goroutines at once.
func Get() Config {
	getMu.Lock()
	defer getMu.Unlock()

	var cfg Config

	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return cfg
}
