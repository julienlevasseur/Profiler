package local

import (
	"bufio"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/julienlevasseur/profiler/config"
	"github.com/julienlevasseur/profiler/pkg/profile"
	yaml "gopkg.in/yaml.v2"
)

// profilePath is the on-disk location of a named local profile. Every caller
// goes through it, so the path is spelled exactly once.
func profilePath(cfg config.Config, profileName string) string {
	return filepath.Join(cfg.ProfilesFolder, "."+profileName+".yml")
}

func getProfileNameFromFilename(filename string) string {
	cfg := config.Get()
	p := strings.Replace(
		filename,
		cfg.ProfilesFolder+"/", "", -1,
	)
	p = strings.Replace(p, ".yml", "", -1)
	p = strings.Replace(p, ".", "", -1)

	return p
}

func fileExist(file string) bool {
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
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

func listFiles(path, extension string) []string {
	var files []string

	files, err := filepath.Glob(path + "/" + extension)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return files
}

func listYamlFiles(path string) []string {
	dotYamlfiles := listFiles(path,
		".*.yml",
	)

	yamlFiles := listFiles(path,
		"*.yaml",
	)

	var files []string
	files = append(files, dotYamlfiles...)
	files = append(files, yamlFiles...)

	return files
}

func loadDotEnvFiles(cfg config.Config) (map[string]string, error) {
	envVars := make(map[string]string)
	for _, thisEnvFile := range listFiles(".", "*.env") {
		if isIgnoredFile(cfg, thisEnvFile) {
			continue
		} else {

			kv, err := parseEnvrc(thisEnvFile)
			if err != nil {
				return map[string]string{}, err
			}
			for k, v := range kv {
				envVars[k] = v
			}
		}
	}

	return envVars, nil
}

func loadDotEnvYaml(cfg config.Config) (map[string]string, error) {
	envVars := make(map[string]string)
	p, err := filepath.Abs(".env.yml")
	if err != nil {
		return map[string]string{}, err
	}
	if fileExist(p) {
		if !isIgnoredFile(cfg, p) {

			kv, err := parseYaml(p)
			if err != nil {
				return map[string]string{}, err
			}

			for k, v := range kv {
				if v != "" {
					envVars[k] = v
				}
			}
		}
	}

	return envVars, nil
}

func loadDotEnvRc(cfg config.Config) (map[string]string, error) {
	envVars := make(map[string]string)
	p, err := filepath.Abs(".envrc")
	if err != nil {
		return map[string]string{}, err
	}
	if fileExist(p) {
		if !isIgnoredFile(cfg, p) {
			kv, err := parseEnvrc(p)
			if err != nil {
				return map[string]string{}, err
			}
			for k, v := range kv {
				envVars[k] = v
			}
		}
	}

	return envVars, nil
}

func GetDotProfiler() (profile.Profile, error) {
	cfg := config.Get()

	envVars := make(map[string]string)

	// check for .profiler file:
	p, err := filepath.Abs(cfg.ProfilerFileName)
	if err != nil {
		return profile.Profile{}, err
	}

	if fileExist(p) {

		kv, err := parseEnvrc(p)
		if err != nil {
			return profile.Profile{}, err
		}
		for k, v := range kv {
			envVars[k] = v
		}
	}

	// check for any .env files:
	for _, thisEnvFile := range listFiles(".", "*.env") {
		kv, err := parseEnvrc(thisEnvFile)
		if err != nil {
			return profile.Profile{}, err
		}
		for k, v := range kv {
			envVars[k] = v
		}
	}
	// check for .env.yml file:
	p, err = filepath.Abs(".env.yml")
	if err != nil {
		return profile.Profile{}, err
	}
	if fileExist(p) {
		kv, err := parseYaml(p)
		if err != nil {
			return profile.Profile{}, err
		}

		for k, v := range kv {
			envVars[k] = v
		}
	}
	// check for .envrc file:
	p, err = filepath.Abs(".envrc")
	if err != nil {
		return profile.Profile{}, err
	}
	if fileExist(p) {
		kv, err := parseEnvrc(p)
		if err != nil {
			return profile.Profile{}, err
		}
		for k, v := range kv {
			envVars[k] = v
		}
	}

	prof := populateProfile(profile.Profile{}, envVars)

	return prof, nil
}

// ignoredFile compare the given filename with the list of ignored files from the config
func isIgnoredFile(cfg config.Config, filename string) bool {
	return slices.Contains(cfg.IgnoredFiles, filename)
}

func parseEnvrc(filename string) (map[string]string, error) {
	envrcVars := make(map[string]string)
	file, err := os.Open(filename)
	if err != nil {
		return map[string]string{}, nil
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		envrc := strings.Split(scanner.Text(), "export ")
		for _, export := range envrc {
			if len(export) == 0 {
				continue
			} else if string(export[0]) == "#" {
				// Since envrc format support comments, ignore comments when found.
				continue
			} else {
				envrcVars[strings.Split(export, "=")[0]] = strings.Replace(
					strings.Split(export, "=")[1], "\"", "", -1,
				)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return map[string]string{}, nil
	}

	return envrcVars, nil
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

func populateProfile(p profile.Profile, kv map[string]string) profile.Profile {
	pName := kv["profile_name"]
	if pName != "" {
		p.Name = kv["profile_name"]
	} else {
		p.KVs = append(
			p.KVs,
			profile.KV{
				Key:   "profile_name",
				Value: "No-Name-Profile",
			},
		)
		p.Name = "No-Name-Profile"
	}

	for k, v := range kv {
		p.KVs = append(p.KVs, profile.KV{Key: k, Value: v})
	}

	return p
}

func AddProfile(args []string) error {
	cfg := config.Get()

	profileName := args[0]
	filePath := profilePath(cfg, profileName)

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
			"The provided variable already exist in %s\n",
			profileName,
		)
		return nil
	}

	err = appendToFile(
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

func GetProfile(profileName string) (profile.Profile, error) {
	cfg := config.Get()
	envVars := make(map[string]string)

	// parse profile file:
	kv, err := parseYaml(profilePath(cfg, profileName))
	if err != nil {
		return profile.Profile{}, err
	}

	for k, v := range kv {
		envVars[k] = v
	}

	// check for any .env files:
	dotEnvVars, err := loadDotEnvFiles(cfg)
	if err != nil {
		return profile.Profile{}, err
	}
	maps.Copy(envVars, dotEnvVars)

	// check for .env.yml file:
	dotEnvYamlVars, err := loadDotEnvYaml(cfg)
	if err != nil {
		return profile.Profile{}, err
	}
	maps.Copy(envVars, dotEnvYamlVars)

	// check for .envrc file:
	dotEnvRcVars, err := loadDotEnvRc(cfg)
	if err != nil {
		return profile.Profile{}, err
	}
	maps.Copy(envVars, dotEnvRcVars)

	p := populateProfile(profile.Profile{}, envVars)

	return p, nil
}

func ListProfiles(path string) ([]string, error) {
	files := listYamlFiles(path)

	var profiles []string
	for _, f := range files {
		profiles = append(profiles, getProfileNameFromFilename(f))
	}

	return profiles, nil
}

func ShowProfile(profileName string) ([]string, error) {
	cfg := config.Get()
	var vars []string

	kvs, err := parseYaml(profilePath(cfg, profileName))
	if err != nil {
		return []string{}, err
	}
	for k := range kvs {
		vars = append(vars, k)
	}

	return vars, nil
}

func UseProfile(args []string) error {
	if len(args) == 0 {
		p, err := GetDotProfiler()
		if err != nil {
			return err
		}

		err = profile.SetEnvironment(p)
		if err != nil {
			return err
		}

	}

	return nil
}
