package cmd

import (
	"errors"
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/julienlevasseur/profiler/pkg/failure"
	"github.com/julienlevasseur/profiler/pkg/local"
)

/*RootCmd root command*/
var RootCmd = &cobra.Command{
	Use:   "profiler",
	Short: "A tool to manage your env vars as profiles.",
	Long: `Profiler is simple tool that allow you to manage your
environment variables.`,
	Run: func(cmd *cobra.Command, args []string) {
		local.UseNoProfile()
	},
}

/*Execute is used in main.go*/
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		failure.ExitOnError(err)
		// fmt.Fprintln(os.Stderr, err)
		// os.Exit(1)
	}
}

func createProfilesFolder() error {
	if _, err := os.Stat(viper.GetString("profilesFolder")); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(viper.GetString("profilesFolder"), os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeDefaultConfigFile(homeFolder, configFile string) {
	defaultConfig := []byte(
		fmt.Sprintf("profilesFolder: %s/.profiles", homeFolder),
	)
	//err := ioutil.WriteFile(configFile, defaultConfig, 0644)
	err := os.WriteFile(configFile, defaultConfig, 0644)
	if err != nil {
		failure.ExitOnError(err)
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

	viper.SetDefault("profilesFolder", fmt.Sprintf("%s/.profiles", home))

	if os.Getenv("PROFILER_CFG") != "" {
		configFile := os.Getenv("PROFILER_CFG")
		viper.SetConfigFile(configFile)
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			writeDefaultConfigFile(home, configFile)
		}
	} else {
		// If there is no .profiler_cfg.yml file (like. for the first execution)
		// let's create a default one.
		configFile := fmt.Sprintf("%s/.profiler_cfg.yml", home)
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			writeDefaultConfigFile(home, configFile)
		}

		viper.SetConfigFile(configFile)
		viper.SetDefault("shell", os.Getenv("SHELL"))
		viper.SetDefault("preserveProfile", true)
		viper.SetDefault("ssmRegion", "us-east-1")
		viper.SetDefault("ssmParameterTier", "Standard")
		viper.SetDefault("consulToken", "")
		viper.SetDefault("consulTokenFile", "")
		viper.SetDefault(
			"repositoryPath",
			fmt.Sprintf(
				"%s/.profiler_repository.json",
				home,
			),
		)
	}

	viper.SetDefault("k8sSwitchNamespace", true)

	viper.AutomaticEnv()
	viper.SetConfigType("yaml")

	err = viper.ReadInConfig()
	if err != nil {
		failure.ExitOnError(err)
		//fmt.Println(err)
	}

	err = createProfilesFolder()
	if err != nil {
		failure.ExitOnError(err)
		//fmt.Println(err)
	}

	//repo := repository.Repository{
	//	Path: viper.GetString("repositoryPath"),
	//}
	//profiles, err := repo.List()
	//if err != nil {
	//	failure.ExitOnError(err)
	//}
	//fmt.Println(profiles)
	//
	//profile, err := repo.Get("aws_perso")
	//if err != nil {
	//	failure.ExitOnError(err)
	//}
	//fmt.Println(profile.ProfileType)
}
