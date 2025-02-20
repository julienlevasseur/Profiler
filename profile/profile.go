package profile

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/julienlevasseur/profiler/config"
	"github.com/spf13/viper"
)

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Profile struct {
	Name string `json:"name"`
	KVs  []KV   `json:"kvs"`
	// Repository string `json:"repository"` a profile is not necessary tied to one repository and may be composed with KVs from different repos
}

type IProfile interface {
	Add(provider string, kvs []KV) error
	Remove() error
	Show() error
	Use() error
	GetDotProfiler() (Profile, error)
}

func New() Profile {
	return Profile{}
}

func (p Profile) Add(repository string, kvs []KV) error {
	return nil
}

func (p Profile) Remove() error {
	return nil
}

func (p Profile) Show() error {
	return nil
}

func (p Profile) Use() error {
	return nil
}

// func (p Profile) GetDotProfiler() (Profile, error) {
// 	envVars := make(map[string]string)
// 	// check for .profiler file:
// 	profile, err := filepath.Abs(".profiler")
// 	if err != nil {
// 		return Profile{}, err
// 	}

// 	if local.fileExist(p) {
// 		p, err := filepath.Abs(".profiler")
// 		if err != nil {
// 			return err
// 		}

// 		kv, err := parseEnvrc(p)
// 		if err != nil {
// 			return err
// 		}
// 		for k, v := range kv {
// 			envVars[k] = v
// 		}
// 	}

// }

func checkForKubernetesNamespace(k, v string) {
	if k == "K8S_NAMESPACE" {
		if v != "" {
			if err := switchKubernetesNamespace(v); err != nil {
				fmt.Println(err)
			}
		}
	}
}

func switchKubernetesNamespace(namespace string) error {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"bash",
		"-c",
		fmt.Sprintf(
			"kubectl config set-context --current --namespace=%s",
			namespace,
		),
	)
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// SetEnvironment read the profilerFile and set a new environment in
// the given shell (exported one if the config doesn't specify one)
// func SetEnvironment(yml map[string]string) error {
func SetEnvironment(profile Profile) error {
	d := []byte("")
	p, err := filepath.Abs(".profiler")
	if err != nil {
		return err
	}

	cfg := config.Get()

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
		if viper.GetBool("k8sSwitchNamespace") {
			checkForKubernetesNamespace(kv.Key, kv.Value)
		}

		os.Setenv(kv.Key, kv.Value)
	}

	if !cfg.PreserveProfile {
		//if !viper.GetBool("preserveProfile") {
		err := os.Remove(".profiler")

		if err != nil {
			return err
		}
	}

	shell := viper.GetString("shell")
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
