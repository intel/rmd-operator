package state

import (
	"reflect"
	"testing"
)

func TestUpdateRmdNodeData(t *testing.T) {
	tcases := []struct {
		name           string
		nodeName       string
		nodeData       RmdNodeData
		expectedStates []string
	}{
		{
			name:     "test case 1 - data added to empty RmdNodeData struct",
			nodeName: "example-node-1",
			nodeData: RmdNodeData{
				RmdNodeList: []string{},
			},
			expectedStates: []string{"example-node-1"},
		},
	}

	for _, tc := range tcases {
		nd := &tc.nodeData
		nd.UpdateRmdNodeData(tc.nodeName)

		if !reflect.DeepEqual(nd.RmdNodeList, tc.expectedStates) {
			t.Errorf("%v failed: Expected: %v, Got: %v\n", tc.name, tc.expectedStates, nd.RmdNodeList)
		}
	}
}

func TestDeleteRmdNodeData(t *testing.T) {
	tcases := []struct {
		name           string
		nodeName       string
		nodeData       RmdNodeData
		expectedStates []string
	}{
		{
			name:     "test case 1 - 2 node state entries, delete one",
			nodeName: "example-node-2",
			nodeData: RmdNodeData{
				RmdNodeList: []string{"example-node-1", "example-node-2"},
			},
			expectedStates: []string{"example-node-1"},
		},
	}

	for _, tc := range tcases {
		nd := &tc.nodeData
		nd.DeleteRmdNodeData(tc.nodeName)
		if !reflect.DeepEqual(nd.RmdNodeList, tc.expectedStates) {
			t.Errorf("%v failed: Expected: %v, Got: %v\n", tc.name, tc.expectedStates, nd.RmdNodeList)
		}
	}
}
