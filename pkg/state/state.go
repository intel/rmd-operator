package state

type RmdNodeData struct {
	RmdNodeList []string
}

// NewRmdNodeData() creates an empty RmdNodeData object if no nodestates are present
// Otherwise, relevant data is extracted from node states and placed in a map
func NewRmdNodeData() *RmdNodeData {
	return &RmdNodeData{
		RmdNodeList: []string{},
	}
}

// UpdateRmdNodeData() adds new data to map or updates existing node data
func (nd *RmdNodeData) UpdateRmdNodeData(nodeName string) {
	for _, node := range nd.RmdNodeList {
		if nodeName == node {
			return
		}
	}
	nd.RmdNodeList = append(nd.RmdNodeList, nodeName)
}

// DeleteRmdNodeData() deletes node data if the corresponding nodestate was deleted
func (nd *RmdNodeData) DeleteRmdNodeData(nodeName string) {
	for index, node := range nd.RmdNodeList {
		if node == nodeName {
			nd.RmdNodeList = append(nd.RmdNodeList[:index], nd.RmdNodeList[index+1:]...)
		}
	}
}
