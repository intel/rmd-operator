package state

import (
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("state")

type RmdNodeData struct {
	RmdNodeList map[string]string
}

// NewRmdNodeData() creates an empty RmdNodeData object if no nodestates are present
// Otherwise, relevant data is extracted from node states and placed in a map
func NewRmdNodeData() *RmdNodeData {
	return &RmdNodeData{
		RmdNodeList: map[string]string{},
	}
}

// UpdateRmdNodeData() adds new data to map or updates existing node data
func (nd *RmdNodeData) UpdateRmdNodeData(nodeName string, namespaceName string) {
	nd.RmdNodeList[nodeName] = namespaceName
}

// DeleteRmdNodeData() deletes node data if the corresponding nodestate was deleted
func (nd *RmdNodeData) DeleteRmdNodeData(nodeName string, namespaceName string) {
	delete(nd.RmdNodeList, nodeName)
}
