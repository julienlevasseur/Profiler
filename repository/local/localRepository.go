package local

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/julienlevasseur/pkg/local"
	"github.com/julienlevasseur/profiler/config"
	"github.com/julienlevasseur/profiler/profile"
	"github.com/spf13/viper"
)

type localRepository struct {
	Name string `mapstructure:"name,omitempty" yaml:"name,omitempty"`
	// Profile localProfile `mapstructure:"profiles,omitempty" yaml:"profile,omitempty"`
}

func NewLocalRepository() *localRepository {
	return &localRepository{
		Name: "local",
	}
}

func getProfileNameFromFilename(filename string) string {
	p := strings.Replace(
		filename,
		viper.GetString("profilesFolder")+"/", "", -1,
	)
	p = strings.Replace(p, ".yml", "", -1)
	p = strings.Replace(p, ".", "", -1)

	return p
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

func (r *localRepository) Add(args []string) error {
	return local.Add(args)
}

func (r *localRepository) IsConfigured() bool {
	return true
}

func (r *localRepository) GetName() string {
	return r.Name
}

func (r *localRepository) List() ([]string, error) {
	pFolder := viper.GetString("profilesFolder")
	files := listYamlFiles(pFolder)

	var profiles []string
	for _, f := range files {
		profiles = append(profiles, getProfileNameFromFilename(f))
	}

	return profiles, nil
}

func (r *localRepository) Get(name string) (profile.Profile, error) {
	cfg := config.Get()
	envVars := make(map[string]string)
	// parse profile file:
	kv, err := parseYaml(
		fmt.Sprintf(
			"%s/.%v.yml",
			cfg.ProfilesFolder,
			name,
		),
	)
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
	dotEnvYamlVars, err := loadDotEnvFiles(cfg)
	if err != nil {
		return profile.Profile{}, err
	}
	maps.Copy(envVars, dotEnvYamlVars)

	// check for .envrc file:
	dotEnvRcVars, err := loadDotEnvFiles(cfg)
	if err != nil {
		return profile.Profile{}, err
	}
	maps.Copy(envVars, dotEnvRcVars)

	p := populateProfile(profile.New(), envVars)

	return p, nil
}

func (r *localRepository) Save(p profile.Profile) error {
	return nil
}

func (r *localRepository) Remove(args []string) error {
	return nil
}

func populateProfile(p profile.Profile, kv map[string]string) profile.Profile {
	if _, ok := kv["profile_name"]; ok {
		p.Name = kv["profile_name"]
	} else {
		p.Name = "No-Name-Profile"
	}

	for k, v := range kv {
		p.KVs = append(p.KVs, profile.KV{Key: k, Value: v})
	}

	return p
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

// parseEnvrc parse the given rc file
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

func GetDotProfiler() (profile.Profile, error) {
	envVars := make(map[string]string)
	// check for .profiler file:
	p, err := filepath.Abs(".profiler")
	if err != nil {
		return profile.Profile{}, err
	}

	if fileExist(p) {
		p, err := filepath.Abs(".profiler")
		if err != nil {
			return profile.Profile{}, err
		}

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

	prof := populateProfile(profile.New(), envVars)

	return prof, nil
}

func (r *localRepository) Show(name string) ([]string, error) {
	return []string{}, nil
}
