package profile

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/viper"

	"github.com/julienlevasseur/profiler/helpers"
)

type KeyValueMap map[string]string

var envVars KeyValueMap
var profilerFile, _ = filepath.Abs(".profiler")
var anyEnvFile = helpers.ListFiles(".", "*.env")
var envFile, _ = filepath.Abs(".env.yml")
var envRcFile, _ = filepath.Abs(".envrc")

func FileExist(file string) bool {
	if _, err := os.Stat(file); err == nil {
		return true
	}

	return false
}

func ParseYaml(filename string) KeyValueMap {
	var y KeyValueMap
	source, err := ioutil.ReadFile((filename))
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(source, &y)
	if err != nil {
		panic(err)
	}

	return y
}

func ParseEnvrc(filename string) KeyValueMap {
	envrcVars := make(map[string]string)
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
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
		panic(err)
	}

	return envrcVars
}

/*SetEnvironment read the profilerFile and set a new environment in
  the given shell (exported one if the config doesn't specify one)*/
func SetEnvironment(yml KeyValueMap) {
	d := []byte("")
	err := ioutil.WriteFile(profilerFile, d, 0644)
	if err != nil {
		panic(err)
	}

	shell := viper.GetString("shell")

	for k, v := range yml {

		file, err := os.OpenFile(profilerFile, os.O_APPEND|os.O_WRONLY, 0644)

		if err != nil {
			panic(err)
		}

		defer file.Close()

		str := fmt.Sprintf("export %s=\"%v\"\n", k, v)
		if _, err = file.WriteString(str); err != nil {
			panic(err)
		}

		os.Setenv(k, v)
	}

	binary, err := exec.LookPath(shell)

	if err != nil {
		panic(err)
	}

	if viper.GetBool("preserveProfile") == false {
		helpers.CleanProfileFile()
	}

	env := os.Environ()
	args := []string{shell}
	err = syscall.Exec(binary, args, env)

	if err != nil {
		panic(err)
	}
}

func GetProfile(profileFolder string, profileName string) KeyValueMap {
	return ParseYaml(
		fmt.Sprintf(
			"%s/.%v.yml",
			profileFolder,
			profileName,
		),
	)
}

/*Use return a map of all the key:value set found in the local accpeted
files, including the given profile*/
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

/*UseNoProfile return a map of all the key:value set found in the local accepted
files*/
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
