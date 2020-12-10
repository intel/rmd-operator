package node

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/intel/rmd-operator/pkg/apis"
	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	"github.com/intel/rmd-operator/pkg/rmd"
	"github.com/intel/rmd-operator/pkg/state"
	rmdCache "github.com/intel/rmd/modules/cache"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strconv"
	"testing"
)

func createReconcileNodeObject(node *corev1.Node) (*ReconcileNode, error) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme

	// Add route Openshift scheme
	if err := apis.AddToScheme(s); err != nil {
		return nil, err
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{node}

	// Register operator types with the runtime scheme.
	s.AddKnownTypes(intelv1alpha1.SchemeGroupVersion)

	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)

	// Create a fake rmd client.
	rmdCl := rmd.NewDefaultOperatorRmdClient()

	// Create an empty RmdNodeData object
	rmdNodeData := &state.RmdNodeData{
		RmdNodeList: []string{},
	}

	// Create a ReconcileNode object with the scheme and fake client.
	r := &ReconcileNode{client: cl, rmdClient: rmdCl, scheme: s, rmdNodeData: rmdNodeData}

	return r, nil

}

func TestNodeControllerReconcile(t *testing.T) {
	tcases := []struct {
		name     string
		node     *corev1.Node
		expected map[string]bool
	}{
		{
			name: "test case 1 - rdtCatLabel present, create all",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-1",
					Labels: map[string]string{
						rdtCatLabel: "true",
					},
				},
			},
			expected: map[string]bool{
				rmdPodNameConst:       true,
				nodeAgentNameConst:    true,
				rmdNodeStateNameConst: true,
			},
		},
		{
			name: "test case 2 - rdtCatLabel present but false, create none",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-2",
					Labels: map[string]string{
						rdtCatLabel: "false",
					},
				},
			},
			expected: map[string]bool{
				rmdPodNameConst:       false,
				nodeAgentNameConst:    false,
				rmdNodeStateNameConst: false,
			},
		},
		{
			name: "test case 3 - rdtCatLabel not present, create none",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-3",
				},
			},
			expected: map[string]bool{
				rmdPodNameConst:       false,
				nodeAgentNameConst:    false,
				rmdNodeStateNameConst: false,
			},
		},
	}

	for _, tc := range tcases {
		rmdPodPath = "../../../build/manifests/rmd-pod.yaml"
		nodeAgentPath = "../../../build/manifests/rmd-node-agent.yaml"

		// Create a ReconcileNode object with the scheme and fake client.
		r, err := createReconcileNodeObject(tc.node)
		if err != nil {
			t.Fatalf("error creating ReconcileNode object: (%v)", err)
		}

		nodeName := tc.node.GetObjectMeta().GetName()

		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: nodeName,
			},
		}

		res, err := r.Reconcile(req)
		if err != nil {
			t.Fatalf("reconcile: (%v)", err)
		}
		namespace := defaultNamespace

		// Initialise all expected value to true
		created := make(map[string]bool)
		created[rmdPodNameConst] = true
		created[nodeAgentNameConst] = true
		created[rmdNodeStateNameConst] = true

		// Check if rmd pod has been created

		rmdPod := &corev1.Pod{}
		rmdPodName := fmt.Sprintf("%s%s", rmdPodNameConst, nodeName)
		rmdPodNamespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      rmdPodName,
		}
		err = r.client.Get(context.TODO(), rmdPodNamespacedName, rmdPod)
		if err != nil {
			created[rmdPodNameConst] = false
		}
		//Check if node agent has been created
		rmdNodeAgentPod := &corev1.Pod{}
		rmdNodeAgentPodName := fmt.Sprintf("%s%s", nodeAgentNameConst, nodeName)
		rmdNodeAgentPodNamespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      rmdNodeAgentPodName,
		}
		err = r.client.Get(context.TODO(), rmdNodeAgentPodNamespacedName, rmdNodeAgentPod)
		if err != nil {
			created[nodeAgentNameConst] = false
		}

		// Check if node state has been created
		rmdNodeState := &intelv1alpha1.RmdNodeState{}
		rmdNodeStateName := fmt.Sprintf("%s%s", rmdNodeStateNameConst, nodeName)
		rmdNodeStatePodNamespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      rmdNodeStateName,
		}
		err = r.client.Get(context.TODO(), rmdNodeStatePodNamespacedName, rmdNodeState)
		if err != nil {
			created[rmdNodeStateNameConst] = false
		}

		//Check the result of reconciliation to make sure it has the desired state.
		if res.Requeue {
			t.Error("reconcile unexpectedly requeued request")
		}

		if !reflect.DeepEqual(created, tc.expected) {
			t.Errorf("Failed: %v - Expected %v, got %v", tc.name, tc.expected, created)
		}

	}
}

func TestAddNodeLabelIfNotPresent(t *testing.T) {
	tcases := []struct {
		name               string
		node               *corev1.Node
		expectedNodeLabels map[string]string
	}{
		{
			name: "test case 1 - no labels present",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-1",
					UID:  "cdaa1644-64f6-4d56-b4a1-c79b03c642cb",
				},
			},
			expectedNodeLabels: map[string]string{
				rmdNodeLabelConst: "cdaa1644-64f6-4d56-b4a1-c79b03c642cb",
			},
		},
		{
			name: "test case 2 - rdtCatLabel present",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-2",
					UID:  "cdaa1644-64f6-4d56-b4a1-c79b03c642cb",
					Labels: map[string]string{
						rdtCatLabel: "true",
					},
				},
			},
			expectedNodeLabels: map[string]string{
				rdtCatLabel:       "true",
				rmdNodeLabelConst: "cdaa1644-64f6-4d56-b4a1-c79b03c642cb",
			},
		},
		{
			name: "test case 3 - rmdNodeLabel present",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-3",
					UID:  "cdaa1644-64f6-4d56-b4a1-c79b03c642cb",
					Labels: map[string]string{
						rmdNodeLabelConst: "cdaa1644-64f6-4d56-b4a1-c79b03c642cb",
					},
				},
			},
			expectedNodeLabels: map[string]string{
				rmdNodeLabelConst: "cdaa1644-64f6-4d56-b4a1-c79b03c642cb",
			},
		},
		{
			name: "test case 4 - rdtCatLabel and rmdNodeLabel both present",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-3",
					UID:  "cdaa1644-64f6-4d56-b4a1-c79b03c642cb",
					Labels: map[string]string{
						rmdNodeLabelConst: "cdaa1644-64f6-4d56-b4a1-c79b03c642cb",
						rdtCatLabel:       "true",
					},
				},
			},
			expectedNodeLabels: map[string]string{
				rmdNodeLabelConst: "cdaa1644-64f6-4d56-b4a1-c79b03c642cb",
				rdtCatLabel:       "true",
			},
		},
	}

	for _, tc := range tcases {
		// Create a ReconcileNode object with the scheme and fake client.
		r, err := createReconcileNodeObject(tc.node)
		if err != nil {
			t.Fatalf("error creating ReconcileNode object: (%v)", err)
		}

		nodeName := tc.node.GetObjectMeta().GetName()
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: nodeName,
			},
		}
		node := &corev1.Node{}
		err = r.client.Get(context.TODO(), req.NamespacedName, node)
		if err != nil {
			t.Fatalf("Could not get node")
		}
		err = r.addNodeLabelIfNotPresent(node)
		if err != nil {
			t.Fatalf("r.addNodeLabelIfNotPresent returned error: (%v)", err)
		}

		nodeLabels := node.GetObjectMeta().GetLabels()
		if !reflect.DeepEqual(nodeLabels, tc.expectedNodeLabels) {
			t.Errorf("Failed: %v - Expected %v, got %v", tc.name, tc.expectedNodeLabels, nodeLabels)
		}

	}
}

func TestCreatePodIfNotPresent(t *testing.T) {
	tcases := []struct {
		name           string
		node           *corev1.Node
		namespacedName types.NamespacedName
		path           string
		podCreated     bool
	}{
		{
			name: "test case 1 - create rmd-pod",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-1",
					UID:  "cdaa1644-64f6-4d56-b4a1-c79b03c642cb",
				},
			},
			namespacedName: types.NamespacedName{
				Name:      "rmd-pod-example-node-1",
				Namespace: "default",
			},
			path:       rmdPodPath,
			podCreated: true,
		},
		{
			name: "test case 2 - create node-agent",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-1",
					UID:  "cdaa1644-64f6-4d56-b4a1-c79b03c642cb",
				},
			},
			namespacedName: types.NamespacedName{
				Name:      "rmd-node-agent-example-node-1",
				Namespace: "default",
			},
			path:       nodeAgentPath,
			podCreated: true,
		},
	}

	for _, tc := range tcases {
		rmdPodPath = "../../../build/manifests/rmd-pod.yaml"
		nodeAgentPath = "../../../build/manifests/rmd-node-agent.yaml"

		// Create a ReconcileNode object with the scheme and fake client.
		r, err := createReconcileNodeObject(tc.node)
		if err != nil {
			t.Fatalf("error creating ReconcileNode object: (%v)", err)
		}

		nodeName := tc.node.GetObjectMeta().GetName()
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: nodeName,
			},
		}
		node := &corev1.Node{}
		err = r.client.Get(context.TODO(), req.NamespacedName, node)
		if err != nil {
			t.Fatalf("Could not get node")
		}

		err = r.createPodIfNotPresent(node, tc.namespacedName, tc.path)
		if err != nil {
			t.Fatalf("r.createPodIfNotPresent returned error: (%v)", err)
		}

		podCreated := true
		pod := &corev1.Pod{}
		err = r.client.Get(context.TODO(), tc.namespacedName, pod)
		if err != nil {
			podCreated = false
		}

		if tc.podCreated != podCreated {
			t.Errorf("Failed: %v - Expected %v, got %v", tc.name, tc.podCreated, podCreated)
		}

	}
}

func TestCreateNodeStateIfNotPresent(t *testing.T) {
	tcases := []struct {
		name             string
		node             *corev1.Node
		namespacedName   types.NamespacedName
		nodeStateCreated bool
	}{
		{
			name: "create rmd node state",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-1.com",
					UID:  "cdaa1644-64f6-4d56-b4a1-c79b03c642cb",
				},
			},
			namespacedName: types.NamespacedName{
				Name:      "rmd-node-state-example-node-1.com",
				Namespace: "default",
			},
			nodeStateCreated: true,
		},
	}

	for _, tc := range tcases {
		// Create a ReconcileNode object with the scheme and fake client.
		r, err := createReconcileNodeObject(tc.node)
		if err != nil {
			t.Fatalf("error creating ReconcileNode object: (%v)", err)
		}

		nodeName := tc.node.GetObjectMeta().GetName()
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: nodeName,
			},
		}
		node := &corev1.Node{}
		err = r.client.Get(context.TODO(), req.NamespacedName, node)
		if err != nil {
			t.Fatalf("Could not get node")
		}

		err = r.createNodeStateIfNotPresent(node, tc.namespacedName)
		if err != nil {
			t.Fatalf("r.createNodeStateIfNotPresent returned error: (%v)", err)
		}

		nodeStateCreated := true
		nodeState := &intelv1alpha1.RmdNodeState{}
		err = r.client.Get(context.TODO(), tc.namespacedName, nodeState)
		if err != nil {
			nodeStateCreated = false
		}

		if tc.nodeStateCreated != nodeStateCreated {
			t.Errorf("Failed: %v - Expected %v, got %v", tc.name, tc.nodeStateCreated, nodeStateCreated)
		}

	}
}

func TestUpdateNodeStatusCapacity(t *testing.T) {
	tcases := []struct {
		name              string
		rmdNode           *corev1.Node
		rmdPod            *corev1.Pod
		response          rmdCache.Infos
		expectedCacheWays int
		expectedError     bool
	}{
		{
			name: "test case 1",
			rmdNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-1.com",
				},
				Status: corev1.NodeStatus{
					Capacity: corev1.ResourceList{
						l3Cache: resource.MustParse("0"),
					},
				},
			},
			rmdPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8443,
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					PodIP: "127.0.0.1",
					PodIPs: []corev1.PodIP{
						{
							IP: "127.0.0.1",
						},
					},
				},
			},

			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						AvailableWays: "7ff",
					},
				},
			},
			expectedCacheWays: 2047,
			expectedError:     false,
		},
		{
			name: "test case 2 - l3 cache resource not listed on node",
			rmdNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-1.com",
				},
				Status: corev1.NodeStatus{
					Capacity: corev1.ResourceList{},
				},
			},
			rmdPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8443,
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					PodIP: "127.0.0.1",
					PodIPs: []corev1.PodIP{
						{
							IP: "127.0.0.1",
						},
					},
				},
			},

			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						AvailableWays: "7ff",
					},
				},
			},
			expectedCacheWays: 2047,
			expectedError:     false,
		},
		{
			name: "test case 3 - changed cache data, but node should not be updated",
			rmdNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-1.com",
				},
				Status: corev1.NodeStatus{
					Capacity: corev1.ResourceList{
						l3Cache: resource.MustParse("4094"),
					},
				},
			},
			rmdPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8443,
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					PodIP: "127.0.0.1",
					PodIPs: []corev1.PodIP{
						{
							IP: "127.0.0.1",
						},
					},
				},
			},

			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						AvailableWays: "7ff", //2047
					},
				},
			},
			expectedCacheWays: 4094, //no update to node capacity
			expectedError:     false,
		},

		{
			name: "test case - cannot contact rmd",
			rmdNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-1.com",
				},
				Status: corev1.NodeStatus{
					Capacity: corev1.ResourceList{
						l3Cache: resource.MustParse("0"),
					},
				},
			},
			rmdPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					PodIP: "127.0.0.1",
					PodIPs: []corev1.PodIP{
						{
							IP: "127.0.0.1",
						},
					},
				},
			},
			expectedCacheWays: 0,
			expectedError:     false,
		},
		{
			name: "test case - no container",
			rmdNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-1.com",
				},
				Status: corev1.NodeStatus{
					Capacity: corev1.ResourceList{
						l3Cache: resource.MustParse("0"),
					},
				},
			},
			rmdPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{},
				Status: corev1.PodStatus{
					PodIP: "127.0.0.1",
					PodIPs: []corev1.PodIP{
						{
							IP: "127.0.0.1",
						},
					},
				},
			},
			expectedCacheWays: 0,
			expectedError:     true,
		},
		{
			name: "test case - no container port",
			rmdNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-node-1.com",
				},
				Status: corev1.NodeStatus{
					Capacity: corev1.ResourceList{
						l3Cache: resource.MustParse("0"),
					},
				},
			},
			rmdPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{},
						},
					},
				},
				Status: corev1.PodStatus{
					PodIP: "127.0.0.1",
					PodIPs: []corev1.PodIP{
						{
							IP: "127.0.0.1",
						},
					},
				},
			},
			expectedCacheWays: 0,
			expectedError:     true,
		},
	}
	for _, tc := range tcases {
		// create a listener with the desired port.
		r, err := createReconcileNodeObject(tc.rmdNode)
		if err != nil {
			t.Fatalf("error creating ReconcileNode object: (%v)", err)
		}

		nodeName := tc.rmdNode.GetObjectMeta().GetName()
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: nodeName,
			},
		}

		address := "127.0.0.1:8443"
		l, err := net.Listen("tcp", address)
		if err != nil {
			t.Fatalf("Failed to create listener")
		}

		ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, err := json.Marshal(tc.response)
			if err == nil {
				fmt.Fprintln(w, string(b[:]))
			}
		}))

		ts.Listener.Close()
		ts.Listener = l

		// Start the server.
		ts.Start()
		err = r.client.Create(context.TODO(), tc.rmdPod)
		if err != nil {
			t.Fatalf("Failed to create pod ")
		}
		rmdPodNamespacedName := types.NamespacedName{
			Name:      tc.rmdPod.GetObjectMeta().GetName(),
			Namespace: tc.rmdPod.GetObjectMeta().GetNamespace(),
		}
		expectedError := false
		err = r.updateNodeStatusCapacity(tc.rmdNode, rmdPodNamespacedName)
		if err != nil {
			expectedError = true
		}

		node := &corev1.Node{}
		err = r.client.Get(context.TODO(), req.NamespacedName, node)
		if err != nil {
			t.Fatalf("Could not get node")
		}
		ways := node.Status.Capacity[l3Cache]
		expectedWays := resource.MustParse(strconv.Itoa(tc.expectedCacheWays))
		if ways != expectedWays || expectedError != tc.expectedError {
			t.Errorf("Failed %v, expected: %v and %v, got %v and %v", tc.name, expectedWays, tc.expectedError, ways, expectedError)
		}

		ts.Close()

	}

}
