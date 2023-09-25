package profile

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	yaml "gopkg.in/yaml.v3"

	"github.com/julienlevasseur/profiler/pkg/ssm"
	"github.com/spf13/viper"
)

type KeyValueMap map[string]string

var profilerFile, _ = filepath.Abs(".profiler")
var anyEnvFile = ListFiles(".", "*.env")
var envFile, _ = filepath.Abs(".env.yml")
var envRcFile, _ = filepath.Abs(".envrc")

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

// AppendToFile append a string to a file.
// It's used by the `add` command to properly append
// new variables to profiles. It also create a profile
// file if it does not exists.
func AppendToFile(filePath, profileName, key, value string) error {

	newProfile := false

	if !FileExist(filePath) {
		newProfile = true

		_, err := os.Create(filePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	f, err := os.OpenFile(filePath,	os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	
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

func FoundInfFile(filePath, match string) (bool, int, error) {

	if _, err := os.Stat(filePath); err != nil {
		_, err = os.Create(filePath)
		if err != nil {
			return false, 0, err
		}
	}

	input, err := ioutil.ReadFile(filePath)
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
	input, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	_, lineNumber, err := FoundInfFile(filePath, match)
	if err != nil {
		fmt.Println(err)
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
	err = ioutil.WriteFile(filePath, []byte(output), 0644)
	if err != nil {
		return err
	}
	return nil
}

// ParseYaml parse the given yaml file
func ParseYaml(filename string) KeyValueMap {
	var y KeyValueMap
	source, err := ioutil.ReadFile((filename))
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
func SetEnvironment(yml KeyValueMap) {
	d := []byte("")
	err := ioutil.WriteFile(profilerFile, d, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for k, v := range yml {

		file, err := os.OpenFile(profilerFile, os.O_APPEND|os.O_WRONLY, 0644)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		defer file.Close()

		str := fmt.Sprintf("export %s=\"%v\"\n", k, v)
		if _, err = file.WriteString(str); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		//if `k8sSwitchNamespace` is activated and the K8S_NAMESPACE env var is set in the profile, profiler will automatically switch namespace to this value.
		if viper.GetBool("k8sSwitchNamespace") {
			checkForKubernetesNamespace(k, v)
		}

		os.Setenv(k, v)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if !viper.GetBool("preserveProfile") {
		err := os.Remove(".profiler")

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	shell := viper.GetString("shell")
	binary, err := exec.LookPath(shell)
	if err != nil {
		fmt.Println(err)
	}

	env := os.Environ()
	args := []string{shell}
	err = syscall.Exec(binary, args, env)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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

	SetEnvironment(envVars)
}

// UseSSMProfile set the environment for the given remote AWS SSM profile
func UseSSMProfile(profileName string) {
	vars, err := ssm.GetProfile(profileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	SetEnvironment(vars)
}

// UseNoProfile return a map of all the key:value set found in the local
// accepted files
func UseNoProfile() {
	envVars := make(map[string]string)
	// check for .profiler file:
	if FileExist(profilerFile) {
		for k, v := range ParseEnvrc(profilerFile) {
			envVars[k] = v
		}
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

	SetEnvironment(envVars)
}

// ShowProfile return a list of keys for the given profile
func ShowProfile(profilesFolder string, profileName string) []string {
	var vars []string

	for k := range GetProfile(profilesFolder, profileName) {
		vars = append(vars, k)
	}

	return vars
}
