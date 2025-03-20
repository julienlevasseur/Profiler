package profile

import (
	"fmt"
	"os/exec"
)

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Profile struct {
	Name string `json:"name"`
	KVs  []KV   `json:"kvs"`
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

// func (p Profile) Add(repository string, kvs []KV) error {
// 	return nil
// }

// func (p Profile) Remove() error {
// 	return nil
// }

// func (p Profile) Show() error {
// 	return nil
// }

// func (p Profile) Use() error {
// 	return nil
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
