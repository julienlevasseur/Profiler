package local

import (
	"github.com/julienlevasseur/profiler/repository"
)

func List(r repository.Repository) []string {
	return r.List()
}
