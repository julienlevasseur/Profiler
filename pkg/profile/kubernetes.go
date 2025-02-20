package profile

import (
	"fmt"
	"os/exec"
)

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
