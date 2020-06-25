package controller

import (
	"github.com/intel/rmd-operator/pkg/controller/rmdnodestate"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, rmdnodestate.Add)
}
