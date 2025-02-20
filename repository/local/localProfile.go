package local

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/julienlevasseur/profiler/config"
	"github.com/julienlevasseur/profiler/profile"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

type localProfile struct {
	Name       string       `json:"name"`
	KVs        []profile.KV `json:"kvs"`
	Repository string       `json:"repository"`
}

//type KeyValueMap map[string]string

func NewLocalProfile() *localProfile {
	return &localProfile{}
}

func parseYaml(filename string) (map[string]string, error) {
	var y map[string]string
	source, err := os.ReadFile((filename))
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(source, &y)
	if err != nil {
		return nil, err
	}

	return y, nil
}

// fileExist return a boolean representing if the given file exists
func fileExist(file string) bool {
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}

// ignoredFile compare the given filename with the list of ignored files from the config
func isIgnoredFile(cfg config.Config, filename string) bool {
	return slices.Contains(cfg.IgnoredFiles, filename)
}

// foundInFile check for a value in the given file and return a boolean
// representing the presence of the value and the line number where the value is
// in the file (if found):
func foundInfFile(filePath, match string) (bool, int, error) {

	if _, err := os.Stat(filePath); err != nil {
		_, err = os.Create(filePath)
		if err != nil {
			return false, 0, err
		}
	}

	input, err := os.ReadFile(filePath)
	if err != nil {
		return false, 0, err
	}

	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, match) && match != "" {
			return true, i, nil
		}
	}

	return false, 0, nil
}

// appendToFile append a string to a file.
// It's used by the `add` command to properly append
// new variables to profiles. It also create a profile
// file if it does not exists.
func appendToFile(filePath, profileName, key, value string) error {

	newProfile := false

	if !fileExist(filePath) {
		newProfile = true

		_, err := os.Create(filePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if newProfile {
		_, err = f.WriteString(
			fmt.Sprintf("profile_name: %s\n", profileName),
		)
	}
	defer f.Close()

	if key != "" && value != "" {
		_, err = f.WriteString(fmt.Sprintf("%s: %s\n", key, value))
	}

	return err
}

func apendProfileNameToFile(filepath, profileName string) error {
	err := appendToFile(
		filepath,
		profileName,
		"profile_name",
		profileName,
	)
	return err
}

func (p *localProfile) Add(repository string, kvs []profile.KV) error {
	// 1. Does only a profile name was provided (create an empty profile) ?
	//    1.1 Does the profile already exists ?
	// 2. Does the KVs provided already exists in the profile ?

	// No KVs provided, let's create an empty profile (with just `profile_name`):
	if len(kvs) < 1 {
		apendProfileNameToFile(
			fmt.Sprintf(
				"%s/.%v.yml",
				viper.GetString("profileFolder"),
				&p.Name,
			),
			p.Name,
		)
	} else {
		// At least one KV as been provided, append them to the file:
		for _, kv := range p.KVs {
			profileNameExists, _, err := foundInfFile(
				fmt.Sprintf(
					"%s/.%v.yml",
					viper.GetString("profileFolder"),
					&p.Name,
				),
				"profile_name",
			)

			if err != nil {
				return err
			}

			// If profile_name is not present in the file, add it:
			if !profileNameExists {
				apendProfileNameToFile(
					fmt.Sprintf(
						"%s/.%v.yml",
						viper.GetString("profileFolder"),
						&p.Name,
					),
					p.Name,
				)
			}

			alreadyExist, _, err := foundInfFile(
				fmt.Sprintf(
					"%s/.%v.yml",
					viper.GetString("profileFolder"),
					&p.Name,
				),
				kv.Key,
			)
			if err != nil {
				return err
			}

			if alreadyExist {
				fmt.Fprintf(
					os.Stderr,
					fmt.Sprintf("The provided variable already exist in %s", p.Name),
					"\n",
				)
				os.Exit(1)
			}

			err = appendToFile(
				viper.GetString("profileFolder"),
				p.Name,
				kv.Key,
				kv.Value,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *localProfile) Remove() error {
	return nil
}

func (p *localProfile) Show() error {
	var vars []string

	kvs, err := parseYaml(
		fmt.Sprintf(
			"%s/.%v.yml",
			viper.GetString("profileFolder"),
			&p.Name,
		),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for k := range kvs {
		vars = append(vars, k)
	}

	fmt.Printf("%s:\n", p.Name)
	// Display each Profile's env var name:
	for _, v := range vars {
		fmt.Printf("- %s\n", v)
	}

	return nil
}

func (p *localProfile) Use() error {
	return nil
}
