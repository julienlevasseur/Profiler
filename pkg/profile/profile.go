package profile

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	yaml "gopkg.in/yaml.v3"

	"github.com/julienlevasseur/profiler/config"
	"github.com/julienlevasseur/profiler/pkg/ssm"
)

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Profile struct {
	Name string `json:"name"`
	KVs  []KV   `json:"kvs"`
}

type KeyValueMap map[string]string

var profilerFile, _ = filepath.Abs(".profiler")
var anyEnvFile = ListFiles(".", "*.env")
var envFile, _ = filepath.Abs(".env.yml")
var envRcFile, _ = filepath.Abs(".envrc")

func MapToProfile(profileName string, m map[string]string) Profile {

	var kvs []KV
	for k, v := range m {
		kvs = append(kvs, KV{
			Key:   k,
			Value: v,
		})
	}

	return Profile{
		Name: profileName,
		KVs:  kvs,
	}
}

func inUseProfileName() string {
	return os.Getenv("profile_name")
}

func requireComposition() bool {
	if inUseProfileName() != "" {
		return true
	}

	return false
}

func ComposeProfile(profiles []Profile) (Profile, error) {
	// for _, p := range profiles {

	// }

	return Profile{}, nil
}

// ListFiles return a list of filenames that match the provided extension
// found in the given folder
func ListFiles(folder string, extension string) []string {
	var files []string

	files, err := filepath.Glob(folder + "/" + extension)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return files
}

// FileExist return a boolean representing if the given file exists
func FileExist(file string) bool {
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}

func FoundInfFile(filePath, match string) (bool, int, error) {

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

// RemoveFromFile remove a line containing the match string from the given file
func RemoveFromFile(filePath, match string) error {
	input, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	_, lineNumber, err := FoundInfFile(filePath, match)
	if err != nil {
		return err
	}

	for i := range lines {
		if i == lineNumber {
			lines[i] = ""
		}
	}
	// Rebuild the file content with newline at the end of each lines:
	output := strings.Join(lines, "\n")
	// Because of the empty line left by the match removal, the following line
	// removes lines that only contains newline char:
	output = strings.Replace(output, "\n\n", "\n", -1)
	// Write the updated content to the profile file:
	err = os.WriteFile(filePath, []byte(output), 0644)
	if err != nil {
		return err
	}
	return nil
}

// ParseYaml parse the given yaml file
func ParseYaml(filename string) KeyValueMap {
	var y KeyValueMap
	source, err := os.ReadFile((filename))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(source, &y)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return y
}

// ParseEnvrc parse the given rc file
func ParseEnvrc(filename string) KeyValueMap {
	envrcVars := make(map[string]string)
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		envrc := strings.Split(scanner.Text(), "export ")
		for _, export := range envrc {
			if len(export) == 0 {
				continue
			} else {
				envrcVars[strings.Split(export, "=")[0]] = strings.Replace(strings.Split(export, "=")[1], "\"", "", -1)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return envrcVars
}

// SetEnvironment read the profilerFile and set a new environment in
// the given shell (exported one if the config doesn't specify one)
// func SetEnvironment(yml map[string]string) error {
func SetEnvironment(profile Profile) error {
	cfg := config.Get()

	d := []byte("")
	p, err := filepath.Abs(cfg.ProfilerFileName)
	if err != nil {
		return err
	}

	err = os.WriteFile(p, d, 0644)
	if err != nil {
		return err
	}

	for _, kv := range profile.KVs {

		file, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0644)

		if err != nil {
			return err
		}

		defer file.Close()

		str := fmt.Sprintf("export %s=\"%v\"\n", kv.Key, kv.Value)
		if _, err = file.WriteString(str); err != nil {
			return err
		}

		//if `k8sSwitchNamespace` is activated and the K8S_NAMESPACE env var is set in the profile, profiler will automatically switch namespace to this value.
		if cfg.K8sSwitchNamespace {
			checkForKubernetesNamespace(kv.Key, kv.Value)
		}

		os.Setenv(kv.Key, kv.Value)
	}

	if !cfg.PreserveProfile {
		//if !viper.GetBool("preserveProfile") {
		err := os.Remove(cfg.ProfilerFileName)

		if err != nil {
			return err
		}
	}

	// shell := viper.GetString("shell")
	shell := cfg.Shell
	binary, err := exec.LookPath(shell)
	if err != nil {
		return err
	}

	env := os.Environ()
	args := []string{shell}
	err = syscall.Exec(binary, args, env)

	if err != nil {
		return err
	}

	return nil
}

// GetProfile retrieve the profile from yaml definition
func GetProfile(profileFolder string, profileName string) KeyValueMap {
	return ParseYaml(
		fmt.Sprintf(
			"%s/.%v.yml",
			profileFolder,
			profileName,
		),
	)
}

// Use set the environment for the given profile
func Use(profilesFolder string, profileName string) {
	envVars := make(map[string]string)
	// parse .profiler file:
	for k, v := range GetProfile(profilesFolder, profileName) {
		envVars[k] = v
	}
	// check for any .env files:
	for _, thisEnvFile := range anyEnvFile {
		for k, v := range ParseEnvrc(thisEnvFile) {
			envVars[k] = v
		}
	}
	// check for .env.yml file:
	if FileExist(envFile) {
		for k, v := range ParseYaml(envFile) {
			envVars[k] = v
		}
	}
	// check for .envrc file:
	if FileExist(envRcFile) {
		for k, v := range ParseEnvrc(envRcFile) {
			envVars[k] = v
		}
	}

	p := MapToProfile(profileName, envVars)
	SetEnvironment(p)
}

// UseSSMProfile set the environment for the given remote AWS SSM profile
func UseSSMProfile(profileName string) {
	vars, err := ssm.GetProfile(profileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	p := MapToProfile(profileName, vars)
	SetEnvironment(p)
}

// ShowProfile return a list of keys for the given profile
func ShowProfile(profilesFolder string, profileName string) []string {
	var vars []string

	for k := range GetProfile(profilesFolder, profileName) {
		vars = append(vars, k)
	}

	return vars
}
