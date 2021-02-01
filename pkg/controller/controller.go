package controller

import (
	"github.com/intel/rmd-operator/pkg/rmd"
	"github.com/intel/rmd-operator/pkg/state"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, *rmd.OperatorRmdClient, *state.RmdNodeData) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, rmdClient *rmd.OperatorRmdClient, rmdNodeData *state.RmdNodeData) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, rmdClient, rmdNodeData); err != nil {
			return err
		}
	}
	return nil
}
