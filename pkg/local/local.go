package local

import (
	"errors"
	"fmt"
	"os"

	"github.com/julienlevasseur/profiler/pkg/profile"
	"github.com/spf13/viper"
)

func AddProfile(args []string) error {

	profileName := args[0]
	filePath := viper.GetString("profilesFolder") + "/." + profileName + ".yml"

	// Local profile
	var key string
	if len(args) <= 2 {
		if len(args) < 2 {
			key = "profile_name"
		} else if len(args) == 2 {
			// If the profile name and only the env var name without a value:
			return errors.New("Please provide a value for the variable")
		}
	} else {
		key = args[1]
	}

	var value string
	if len(args) < 2 {
		value = profileName
	} else {
		value = args[2]
	}

	alreadyExist, _, err := profile.FoundInfFile(
		filePath,
		key,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if alreadyExist {
		fmt.Fprintf(
			os.Stderr,
			fmt.Sprintf("The provided variable already exist in %s", profileName),
			"\n",
		)
		return nil
	}

	err = profile.AppendToFile(
		filePath,
		profileName,
		key,
		value,
	)

	if err != nil {
		return err
	}

	return nil
}
