package state

import (
	"context"
	"fmt"
	"os"

	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

//var rmdPodPath = "/rmd-manifests/rmd-pod.yaml"
var log = logf.Log.WithName("state")

type RmdNodeData struct {
	StateMap map[string]string
}

var rmdStateList = map[string]string{}

// NewRmdNodeData() creates an empty RmdNodeData object if no nodestates are present
// Otherwise, relevant data is extracted from node states and placed in a map
func NewRmdNodeData() *RmdNodeData {
	// Create client
	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		log.Error(err, "failed to create client")
		os.Exit(1)
	}

	// List RmdNodeStates
	rmdNodeStates := &intelv1alpha1.RmdNodeStateList{}
	err = cl.List(context.TODO(), rmdNodeStates)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("No RMD Node States present, Return empty RmdNodeData")
			return &RmdNodeData{}
		}
		log.Error(err, "failed to list RMD Node States")
		return &RmdNodeData{}
	}

	// Assign relevant node state information into map
	for _, rmdNodeState := range rmdNodeStates.Items {
		rmdStateList[rmdNodeState.Spec.Node] = rmdNodeState.GetObjectMeta().GetNamespace()
	}
	log.Info("RMD Node States found, Return RmdNodeData with state list")
	log.Info(fmt.Sprintf("State List: %v", rmdStateList))
	return &RmdNodeData{
		StateMap: rmdStateList,
	}
}

// UpdateRmdNodeData() adds new data to map or updates existing node data
func (nd *RmdNodeData) UpdateRmdNodeData(nodeName string, namespaceName string) {
	rmdStateList[nodeName] = namespaceName
	nd.StateMap = rmdStateList
}

// DeleteRmdNodeData() deletes node data if the corresponding nodestate was deleted
func (nd *RmdNodeData) DeleteRmdNodeData(nodeName string, namespaceName string) {
	delete(rmdStateList, nodeName)
	nd.StateMap = rmdStateList
}
