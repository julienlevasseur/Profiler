package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	homedir "github.com/mitchellh/go-homedir"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/julienlevasseur/profiler/pkg/profile"
)

/*RootCmd root command*/
var RootCmd = &cobra.Command{
	Use:   "profiler",
	Short: "A tool to manage your env vars as profiles.",
	Long: `Profiler is simple tool that allow you to manage your
environment variables.`,
	Run: func(cmd *cobra.Command, args []string) {
		profile.UseNoProfile()
	},
}

/*Execute is used in main.go*/
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeDefaultConfigFile(homeFolder, configFile string) {
	defaultConfig := []byte(
		fmt.Sprintf("profilerFolder: %s/.profiles", homeFolder),
	)
	err := ioutil.WriteFile(configFile, defaultConfig, 0644)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(InitConfig)
}

// InitConfig manage configuration
func InitConfig() {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var configFile string

	if os.Getenv("PROFILER_CFG") != "" {
		configFile = os.Getenv("PROFILER_CFG")
		viper.SetConfigFile(configFile)
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			writeDefaultConfigFile(home, configFile)
		}
	} else {
		// If there is no .profiler_cfg.yml file (like. for the first execution)
		// let's create a default one.
		configFile = fmt.Sprintf("%s/.profiler_cfg.yml", home)
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			writeDefaultConfigFile(home, configFile)
		}
	}

	viper.SetConfigType("yaml")
	viper.SetConfigFile(configFile)
	viper.AutomaticEnv()

	viper.SetDefault("preserveProfile", true)
	viper.SetDefault("ssmRegion", "us-east-1")
	viper.SetDefault("ssmParameterTier", "Standard")
	viper.SetDefault("consulAddress", "127.0.0.1:8500")
	viper.SetDefault("consulToken", "")
	viper.SetDefault("consulTokenFile", "")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Println(err)
	}
}
