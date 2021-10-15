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

	"github.com/julienlevasseur/profiler/pkg/ssm"
	"github.com/spf13/viper"
)

type KeyValueMap map[string]string

var envVars KeyValueMap
var profilerFile, _ = filepath.Abs(".profiler")
var anyEnvFile = ListFiles(".", "*.env")
var envFile, _ = filepath.Abs(".env.yml")
var envRcFile, _ = filepath.Abs(".envrc")

/*ListFiles return a list of filenames that match the provided extension
* found in the given folder */
func ListFiles(folder string, extension string) []string {
	var files []string

	files, err := filepath.Glob(folder + "/" + extension)
	if err != nil {
		panic(err)
	}
	return files
}

/*FileExist return a boolean representing if the given file exists*/
func FileExist(file string) bool {
	if _, err := os.Stat(file); err == nil {
		return true
	}

	return false
}

/*AppendToFile append a string to a file.
It's used by the `add` command to properly append
new variables to profiles. It also create a profile
file if it does not exists.*/
func AppendToFile(filePath string, value string) error {
	f, err := os.OpenFile(filePath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	defer f.Close()
	_, err = f.WriteString(value)

	return err
}

/*ParseYaml parse the given yaml file */
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

/*ParseEnvrc parse the given rc file */
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
* the given shell (exported one if the config doesn't specify one)
 */
func SetEnvironment(yml KeyValueMap) {
	d := []byte("")
	err := ioutil.WriteFile(profilerFile, d, 0644)
	if err != nil {
		panic(err)
	}

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

	if err != nil {
		panic(err)
	}

	if viper.GetBool("preserveProfile") == false {
		err := os.Remove(".profiler")

		if err != nil {
			panic(err)
		}
	}

	shell := viper.GetString("shell")
	binary, err := exec.LookPath(shell)

	env := os.Environ()
	args := []string{shell}
	err = syscall.Exec(binary, args, env)

	if err != nil {
		panic(err)
	}
}

/*GetProfile retrieve the profile from yaml definition*/
func GetProfile(profileFolder string, profileName string) KeyValueMap {
	return ParseYaml(
		fmt.Sprintf(
			"%s/.%v.yml",
			profileFolder,
			profileName,
		),
	)
}

/*Use set the environment for the given profile*/
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

/*UseSSMProfile set the environment for the given remote AWS SSM profile*/
func UseSSMProfile(profileName string) {
	vars, err := ssm.GetProfile(profileName)
	if err != nil {
		panic(err)
	}

	SetEnvironment(vars)
}

/*UseNoProfile return a map of all the key:value set found in the local accepted
* files
 */
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

/*ShowProfile return a list of keys for the given profile */
func ShowProfile(profilesFolder string, profileName string) []string {
	var vars []string

	for k := range GetProfile(profilesFolder, profileName) {
		vars = append(vars, k)
	}

	return vars
}
