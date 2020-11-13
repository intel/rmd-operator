package state

import (
	"reflect"
	"testing"
)

func TestUpdateRmdNodeData(t *testing.T) {
	tcases := []struct {
		name           string
		nodeName       string
		namespaceName  string
		nodeData       RmdNodeData
		expectedStates map[string]string
	}{
		{
			name:          "test case 1 - namespace updated for particular node",
			nodeName:      "example-node-1",
			namespaceName: "example-name-space",
			nodeData: RmdNodeData{
				StateMap: map[string]string{
					"example-node-1": "default",
				},
			},
			expectedStates: map[string]string{
				"example-node-1": "example-name-space",
			},
		},

		{
			name:          "test case 2 - data added to empty RmdNodeData struct",
			nodeName:      "example-node-1",
			namespaceName: "default",
			nodeData: RmdNodeData{
				StateMap: map[string]string{},
			},
			expectedStates: map[string]string{
				"example-node-1": "default",
			},
		},
	}

	for _, tc := range tcases {
		nd := &tc.nodeData
		nd.UpdateRmdNodeData(tc.nodeName, tc.namespaceName)

		if !reflect.DeepEqual(nd.StateMap, tc.expectedStates) {
			t.Errorf("%v failed: Expected: %v, Got: %v\n", tc.name, tc.expectedStates, nd.StateMap)
		}
	}
}

func TestDeleteRmdNodeData(t *testing.T) {
	tcases := []struct {
		name           string
		nodeName       string
		namespaceName  string
		nodeData       RmdNodeData
		expectedStates map[string]string
	}{
		{
			name:          "test case 1 - 2 node state entries, delete one",
			nodeName:      "example-node-2",
			namespaceName: "default",
			nodeData: RmdNodeData{
				StateMap: map[string]string{
					"example-node-1": "default",
					"example-node-2": "default",
				},
			},
			expectedStates: map[string]string{
				"example-node-1": "default",
			},
		},
	}

	for _, tc := range tcases {
		nd := &tc.nodeData
		nd.DeleteRmdNodeData(tc.nodeName, tc.namespaceName)
		if !reflect.DeepEqual(nd.StateMap, tc.expectedStates) {
			t.Errorf("%v failed: Expected: %v, Got: %v\n", tc.name, tc.expectedStates, nd.StateMap)
		}
	}
}
