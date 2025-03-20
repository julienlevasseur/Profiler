package config

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

type Config struct {
	ProfilesFolder        string   `mapstructure:"profilesFolder,omitempty" yaml:"profilesFolder,omitempty"`
	ProfilerFileName      string   `mapstructure:"profilerFileName,omitempty" yaml:"profilerFileName,omitempty"`
	LocalProfiles         []string `mapstructure:"localProfiles,omitempty" yaml:"localProfiles,omitempty"`
	PreserveProfile       bool     `mapstructure:"preserve_profile,omitempty" yaml:"preserve_profile,omitempty"`
	Shell                 string   `mapstracture:"shell,omitempty" yaml:"shell,omitempty"`
	ShowRepoSourceInList  bool     `mapstructure:"showRepoSourceInList,omitempty" yaml:"showRepoSourceInList,omitempty"`
	SupportedRepositories []string `mapstructure:"supportedRepositories,omitempty" yaml:"supportedRepositories,omitempty"`
	SSMProfiles           []string `mapstructure:"ssmProfiles,omitempty" yaml:"ssmProfiles,omitempty"`
	// TODO:  Is ConsulProfiles used ?
	ConsulProfiles        []string `mapstructure:"consulProfiles,omitempty" yaml:"consulProfiles,omitempty"`
	AWS_ACCESS_KEY_ID     string   `mapstructure:"aws_access_key_id,omitempty" yaml:"aws_access_key_id,omitempty"`
	AWS_SECRET_ACCESS_KEY string   `mapstructure:"aws_secret_access_key,omitempty" yaml:"aws_secret_access_key,omitempty"`
	AWS_SESSION_TOKEN     string   `mapstructure:"aws_session_token,omitempty" yaml:"aws_session_token,omitempty,omitempty"`
	ConsulAddress         string   `mapstructure:"consul_address,omitempty" yaml:"consulAddress,omitempty"`
	ConsulToken           string   `mapstructure:"consul_token,omitempty" yaml:"consulToken,omitempty"`
	ConsulTokenFile       string   `mapstructure:"consul_token_file,omitempty" yaml:"consulTokenFile,omitempty"`
	ConsulProfilesPath    string   `mapstructure:"consul_profiles_path,omitempty" yaml:"consulProfilesPath,omitempty"`
	VaultAddress          string   `mapstructure:"vault_address,omitempty" yaml:"vaultAddress,omitempty"`
	VaultToken            string   `mapstructure:"vault_token,omitempty" yaml:"vaultToken,omitempty"`
	VaultProfilesPath     string   `mapstructure:"vault_profiles_path,omitempty" yaml:"vaultProfilesPath,omitempty"`
	IgnoredFiles          []string `mapstructure:"ignored_files" yaml:"ignoredFiles"`
}

func InitCfg() {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var cfgFilePath string
	if os.Getenv("PROFILER_CFG") != "" {
		cfgFilePath = os.Getenv("PROFILER_CFG")
	} else {
		cfgFilePath = home
	}

	var cfg Config

	viper.AutomaticEnv()
	viper.AddConfigPath(cfgFilePath)
	viper.SetConfigName(".profiler_cfg")
	viper.SetConfigType("yml")

	// viper.BindEnv("consulAddress")

	viper.SetEnvPrefix("profiler") // will be uppercased automatically

	//viper.SetDefault("profilesFolder", fmt.Sprintf("%s/.profiles", home))

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

	if err := viper.BindEnv("aws_access_key_id"); err != nil {
		fmt.Println(err)
	}

	if err := viper.BindEnv("aws_secret_access_key"); err != nil {
		fmt.Println(err)
	}

	if err := viper.BindEnv("aws_session_token"); err != nil {
		fmt.Println(err)
	}

	if err := viper.BindEnv("ConsulAddress"); err != nil {
		fmt.Println(err)
	}

	if err := viper.BindEnv("consul_token"); err != nil {
		fmt.Println(err)
	}

	if err := viper.BindEnv("consul_token_file"); err != nil {
		fmt.Println(err)
	}

	viper.SetDefault("consulProfilesPath", "profiler")

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

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Println(err)
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Println(err)
	}
}

func Get() Config {
	return Config{
		ProfilesFolder:        viper.GetString("ProfilesFolder"),
		ProfilerFileName:      viper.GetString("profilerFileName"),
		LocalProfiles:         viper.GetStringSlice("LocalProfiles"),
		PreserveProfile:       viper.GetBool("PreserveProfile"),
		Shell:                 viper.GetString("shell"),
		ShowRepoSourceInList:  viper.GetBool("ShowRepoSourceInList"),
		SupportedRepositories: viper.GetStringSlice("SupportedRepositories"),
		SSMProfiles:           viper.GetStringSlice("SSMProfiles"),
		ConsulProfiles:        viper.GetStringSlice("ConsulProfiles"),
		AWS_ACCESS_KEY_ID:     viper.GetString("AWS_ACCESS_KEY_ID"),
		AWS_SECRET_ACCESS_KEY: viper.GetString("AWS_SECRET_ACCESS_KEY"),
		AWS_SESSION_TOKEN:     viper.GetString("AWS_SESSION_TOKEN"),
		ConsulAddress:         viper.GetString("ConsulAddress"),
		ConsulToken:           viper.GetString("ConsulToken"),
		ConsulProfilesPath:    viper.GetString("consulProfilesPath"),
		ConsulTokenFile:       viper.GetString("ConsulTokenFile"),
		VaultAddress:          viper.GetString("VaultAddress"),
		VaultToken:            viper.GetString("VaultToken"),
		VaultProfilesPath:     viper.GetString("VaultProfilesPath"),
		IgnoredFiles:          viper.GetStringSlice("ignoredFiles"),
	}
}
