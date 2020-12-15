package controller

import (
	"github.com/intel/rmd-operator/pkg/controller/rmdconfig"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, rmdconfig.Add)
}
