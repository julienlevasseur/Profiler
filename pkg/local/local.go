package local

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v3"
)

var profilerFile, _ = filepath.Abs(".profiler")
var anyEnvFile = ListFiles(".", "*.env")
var envFile, _ = filepath.Abs(".env.yml")
var envRcFile, _ = filepath.Abs(".envrc")

type LocalProfile struct {
	Name string
	KVs  map[string]string
}

func (lp LocalProfile) Add() error {
	filePath := viper.GetString("profilesFolder") + "/." + lp.Name + ".yml"

	if len(lp.KVs) == 0 {
		// Only the profile name was provided, a profile will be created with the
		// only profile_name: ${profile_name} entry:
		return createEmptyProfile(filePath, lp.Name)
	}

	for k, v := range lp.KVs {
		ok, _, err := foundInfFile(filePath, k)
		if err != nil {
			return err
		}

		if !ok {
			// Does not exists yet in the file, add it:
			err = appendToFile(
				filePath,
				k,
				v,
			)

			if err != nil {
				return err
			}
		} else {
			// Already exists in the file, does it has the same value ?
			err := manageExistingVariable(filePath, k, v)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (lp LocalProfile) Exists() (bool, error) {
	if _, err := os.Stat(
		fmt.Sprintf(
			"%s/.%s.yml",
			viper.GetString("profilesFolder"),
			lp.Name,
		),
	); err != nil {
		return false, err
	}

	return true, nil
}

func (lp LocalProfile) Show() error {

	for k := range parseYaml(
		fmt.Sprintf(
			"%s/.%s.yml",
			viper.GetString("profilesFolder"),
			lp.Name,
		),
	) {
		fmt.Println(k)
	}
	return nil
}

func (lp LocalProfile) Use() {
	envVars := make(map[string]string)
	// parse .profiler file:
	for k, v := range getProfile(viper.GetString("profilesFolder"), lp.Name) {
		envVars[k] = v
	}
	// check for any .env files:
	for _, thisEnvFile := range anyEnvFile {
		for k, v := range parseEnvrc(thisEnvFile) {
			envVars[k] = v
		}
	}
	// check for .env.yml file:
	if fileExists(envFile) {
		for k, v := range parseYaml(envFile) {
			envVars[k] = v
		}
	}
	// check for .envrc file:
	if fileExists(envRcFile) {
		for k, v := range parseEnvrc(envRcFile) {
			envVars[k] = v
		}
	}

	SetEnvironment(envVars)
}

func createEmptyProfile(filePath, profileName string) error {
	return appendToFile(
		filePath,
		"profile_name",
		profileName,
	)
}

func manageExistingVariable(filePath, k, v string) error {
	// Search for "key: value" combination to check if it already exists.
	ok, _, err := foundInfFile(filePath, fmt.Sprintf("%v: %v", k, v))

	if err != nil {
		return err
	}

	if ok {
		// It has the same value, no update required.
		return nil
	} else {
		// If not, update the entry
		err := removeFromFile(filePath, k)

		if err != nil {
			return err
		}

		err = appendToFile(
			filePath,
			k,
			v,
		)

		if err != nil {
			return err
		}
	}

	return nil
}

// getProfile retrieve the profile from yaml definition
func getProfile(profileFolder string, profileName string) map[string]string {
	return parseYaml(
		fmt.Sprintf(
			"%s/.%v.yml",
			profileFolder,
			profileName,
		),
	)
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

func parseYaml(filename string) map[string]string {
	var y map[string]string
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

// parseEnvrc parse the given rc file
func parseEnvrc(filename string) map[string]string {
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

// FileExists return a boolean representing if the given file exists
func fileExists(file string) bool {
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}

// appendToFile append a string to a file.
// It's used by the Create method to properly append new variables to local
// profiles. It also creates a profile file if it does not exists.
func appendToFile(filePath, key, value string) error {

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	if key != "" && value != "" {
		_, err = f.WriteString(fmt.Sprintf("%s: %s\n", key, value))
	}

	return err
}

// foundInfFile check for the presence of a key in the given file.
func foundInfFile(filePath, match string) (bool, int, error) {
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

// removeFromFile remove a line containing the match string from the given file
func removeFromFile(filePath, match string) error {
	fmt.Println("In removeFromFile")
	input, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	_, lineNumber, err := foundInfFile(filePath, match)
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
	err = os.WriteFile(filePath, []byte(output), 0644)
	if err != nil {
		return err
	}
	return nil
}

// UseNoProfile return a map of all the key:value set found in the local
// accepted files
func UseNoProfile() {
	envVars := make(map[string]string)
	// check for .profiler file:
	if fileExists(profilerFile) {
		for k, v := range parseEnvrc(profilerFile) {
			envVars[k] = v
		}
	}
	// check for any .env files:
	for _, thisEnvFile := range anyEnvFile {
		for k, v := range parseEnvrc(thisEnvFile) {
			envVars[k] = v
		}
	}
	// check for .env.yml file:
	if fileExists(envFile) {
		for k, v := range parseYaml(envFile) {
			envVars[k] = v
		}
	}
	// check for .envrc file:
	if fileExists(envRcFile) {
		for k, v := range parseYaml(envRcFile) {
			envVars[k] = v
		}
	}

	SetEnvironment(envVars)
}

// SetEnvironment read the profilerFile and set a new environment in
// the given shell (exported one if the config doesn't specify one)
func SetEnvironment(envVars map[string]string) {
	d := []byte("")
	err := os.WriteFile(profilerFile, d, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for k, v := range envVars {

		file, err := os.OpenFile(profilerFile, os.O_APPEND|os.O_WRONLY, 0644)
		defer file.Close()

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		str := fmt.Sprintf("export %s=\"%v\"\n", k, v)
		if _, err = file.WriteString(str); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		//if `k8sSwitchNamespace` is activated and the K8S_NAMESPACE env var is set in the profile, profiler will automatically switch namespace to this value.
		//if viper.GetBool("k8sSwitchNamespace") {
		//	profile.checkForKubernetesNamespace(k, v)
		//}

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
