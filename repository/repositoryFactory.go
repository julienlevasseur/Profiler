package repository

import (
	"errors"

	"github.com/julienlevasseur/profiler/repository/consul"
	"github.com/julienlevasseur/profiler/repository/local"
	"github.com/julienlevasseur/profiler/repository/ssm"
	"github.com/julienlevasseur/profiler/repository/vault"
)

const (
	localRepoType  = "local"
	SSMRepoType    = "ssm"
	consulRepoType = "consul"
	vaultRepoType  = "vault"
)

func GetRepository(repoType string) (IRepository, error) {
	if repoType == localRepoType {
		return local.NewLocalRepository(), nil
	} else if repoType == SSMRepoType {
		return ssm.NewSSMRepository(), nil
	} else if repoType == consulRepoType {
		return consul.NewConsulRepository(), nil
	} else if repoType == vaultRepoType {
		return vault.NewVaultRepository(), nil
	}

	return nil, errors.New("no matching Repository type")
}
