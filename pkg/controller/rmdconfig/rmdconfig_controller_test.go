package rmdconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/intel/rmd-operator/pkg/apis"
	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	"github.com/intel/rmd-operator/pkg/rmd"
	"github.com/intel/rmd-operator/pkg/state"
	rmdCache "github.com/intel/rmd/modules/cache"
	appsv1 "k8s.io/api/apps/v1"
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

func createReconcileRmdConfigObject(rmdConfig *intelv1alpha1.RmdConfig) (*ReconcileRmdConfig, error) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme

	// Add route Openshift scheme
	if err := apis.AddToScheme(s); err != nil {
		return nil, err
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{rmdConfig}

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
	r := &ReconcileRmdConfig{client: cl, rmdClient: rmdCl, scheme: s, rmdNodeData: rmdNodeData}

	return r, nil

}

func TestRmdConfigControllerReconcile(t *testing.T) {
	tcases := []struct {
		name                 string
		rmdConfig            *intelv1alpha1.RmdConfig
		nodeList             *corev1.NodeList
		podList              *corev1.PodList
		rmdNodeStatesCreated int
		rmdDSCreated         bool
		nodeAgentDSCreated   bool
		expectedRmdConfig    *intelv1alpha1.RmdConfig
	}{
		{
			name: "test case 1 - single node with nodeselector label",
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
				Spec: intelv1alpha1.RmdConfigSpec{
					RmdImage: "rmd:latest",
					RmdNodeSelector: map[string]string{
						rdtCatLabel: "true",
					},
				},
			},
			nodeList: &corev1.NodeList{
				Items: []corev1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "example-node-1",
							Labels: map[string]string{
								rdtCatLabel: "true",
							},
						},
					},
				},
			},
			podList: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-abcde",
							Namespace: defaultNamespace,
							Labels: map[string]string{
								"name": "rmd-pod",
							},
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
							NodeName: "example-node-1",
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
				},
			},

			rmdNodeStatesCreated: 1,
			rmdDSCreated:         true,
			nodeAgentDSCreated:   true,
			expectedRmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
				Spec: intelv1alpha1.RmdConfigSpec{
					RmdImage: "rmd:latest",
					RmdNodeSelector: map[string]string{
						rdtCatLabel: "true",
					},
				},
				Status: intelv1alpha1.RmdConfigStatus{
					Nodes: []string{"example-node-1"},
				},
			},
		},
		{
			name: "test case 2 - single node with RDT L3 CAT label. No RmdNodeSelector defined - default set to RDT L3 CAT label",
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
				Spec: intelv1alpha1.RmdConfigSpec{
					RmdImage: "rmd:latest",
				},
			},
			nodeList: &corev1.NodeList{
				Items: []corev1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "example-node-1",
							Labels: map[string]string{
								rdtCatLabel: "true",
							},
						},
					},
				},
			},
			podList: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-abcde",
							Namespace: defaultNamespace,
							Labels: map[string]string{
								"name": "rmd-pod",
							},
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
							NodeName: "example-node-1",
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
				},
			},

			rmdNodeStatesCreated: 1,
			rmdDSCreated:         true,
			nodeAgentDSCreated:   true,
			expectedRmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
				Spec: intelv1alpha1.RmdConfigSpec{
					RmdImage: "rmd:latest",
					RmdNodeSelector: map[string]string{
						rdtCatLabel: "true",
					},
				},
				Status: intelv1alpha1.RmdConfigStatus{
					Nodes: []string{"example-node-1"},
				},
			},
		},

		{
			name: "test case 2 - 3 nodes, 1 with all labels, 2 with some",
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
				Spec: intelv1alpha1.RmdConfigSpec{
					RmdImage: "rmd:latest",
					RmdNodeSelector: map[string]string{
						rdtCatLabel: "true",
						"feature.node.kubernetes.io/cpu-sstbf.enabled": "true",
					},
				},
			},
			nodeList: &corev1.NodeList{
				Items: []corev1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "example-node-1",
							Labels: map[string]string{
								rdtCatLabel: "true",
								"feature.node.kubernetes.io/cpu-sstbf.enabled": "true",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "example-node-2",
							Labels: map[string]string{
								rdtCatLabel: "true",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "example-node-3",
							Labels: map[string]string{
								"feature.node.kubernetes.io/cpu-sstbf.enabled": "true",
							},
						},
					},
				},
			},
			podList: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-abcde",
							Namespace: defaultNamespace,
							Labels: map[string]string{
								"name": "rmd-pod",
							},
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
							NodeName: "example-node-1",
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
				},
			},

			rmdNodeStatesCreated: 1,
			rmdDSCreated:         true,
			nodeAgentDSCreated:   true,
			expectedRmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
				Spec: intelv1alpha1.RmdConfigSpec{
					RmdImage: "rmd:latest",
					RmdNodeSelector: map[string]string{
						rdtCatLabel: "true",
					},
				},
				Status: intelv1alpha1.RmdConfigStatus{
					Nodes: []string{"example-node-1"},
				},
			},
		},
		{
			name: "test case 4 - 3 nodes, 2 with RmdNodeSelector label",
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
				Spec: intelv1alpha1.RmdConfigSpec{
					RmdImage: "rmd:latest",
					RmdNodeSelector: map[string]string{
						rdtCatLabel: "true",
					},
				},
			},
			nodeList: &corev1.NodeList{
				Items: []corev1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "example-node-1",
							Labels: map[string]string{
								rdtCatLabel: "true",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "example-node-2",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "example-node-3",
							Labels: map[string]string{
								rdtCatLabel: "true",
							},
						},
					},
				},
			},
			podList: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-abcde",
							Namespace: defaultNamespace,
							Labels: map[string]string{
								"name": "rmd-pod",
							},
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
							NodeName: "example-node-1",
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
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-fghij",
							Namespace: defaultNamespace,
							Labels: map[string]string{
								"name": "rmd-pod",
							},
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
							NodeName: "example-node-3",
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
				},
			},

			rmdNodeStatesCreated: 2,
			rmdDSCreated:         true,
			nodeAgentDSCreated:   true,
			expectedRmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
				Spec: intelv1alpha1.RmdConfigSpec{
					RmdImage: "rmd:latest",
					RmdNodeSelector: map[string]string{
						rdtCatLabel: "true",
					},
				},
				Status: intelv1alpha1.RmdConfigStatus{
					Nodes: []string{"example-node-1", "example-node-3"},
				},
			},
		},
	}

	for _, tc := range tcases {
		rmdDaemonSetPath = "../../../build/manifests/rmd-ds.yaml"
		nodeAgentDaemonSetPath = "../../../build/manifests/rmd-node-agent-ds.yaml"

		// Create a ReconcileNode object with the scheme and fake client.
		r, err := createReconcileRmdConfigObject(tc.rmdConfig)
		if err != nil {
			t.Fatalf("error creating ReconcileNode object: (%v)", err)
		}
		for i := range tc.nodeList.Items {
			err = r.client.Create(context.TODO(), &tc.nodeList.Items[i])
			if err != nil {
				t.Fatalf("Failed to create dummy node")
			}
		}
		for i := range tc.podList.Items {
			err = r.client.Create(context.TODO(), &tc.podList.Items[i])
			if err != nil {
				t.Fatalf("Failed to create dummy rmd pod")
			}
		}

		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      tc.rmdConfig.GetObjectMeta().GetName(),
				Namespace: tc.rmdConfig.GetObjectMeta().GetNamespace(),
			},
		}

		res, err := r.Reconcile(req)
		if err != nil {
			t.Fatalf("reconcile: (%v)", err)
		}

		// Initialise all expected value to true
		rmdCreated := true
		nodeAgentCreated := true

		// Check if rmd pod has been created

		rmdDS := &appsv1.DaemonSet{}
		rmdNamespacedName := types.NamespacedName{
			Namespace: defaultNamespace,
			Name:      rmdConst,
		}
		err = r.client.Get(context.TODO(), rmdNamespacedName, rmdDS)
		if err != nil {
			rmdCreated = false
		}
		//Check if node agent has been created
		rmdNodeAgentDS := &appsv1.DaemonSet{}
		rmdNodeAgentDSNamespacedName := types.NamespacedName{
			Namespace: defaultNamespace,
			Name:      nodeAgentNameConst,
		}
		err = r.client.Get(context.TODO(), rmdNodeAgentDSNamespacedName, rmdNodeAgentDS)
		if err != nil {
			nodeAgentCreated = false
		}

		// Check if node state has been created
		rmdNodeStateList := &intelv1alpha1.RmdNodeStateList{}
		err = r.client.List(context.TODO(), rmdNodeStateList)
		if err != nil {
			t.Fatalf("Could not list rmd node states")
		}
		if tc.rmdNodeStatesCreated != len(rmdNodeStateList.Items) {
			t.Errorf("Failed: %v - Expected %v rmdNodeStates, got %v", tc.name, tc.rmdNodeStatesCreated, len(rmdNodeStateList.Items))
		}

		//Check the result of reconciliation to make sure it has the desired state.
		if res.Requeue {
			t.Error("reconcile unexpectedly requeued request")
		}

		if rmdCreated != tc.rmdDSCreated {
			t.Errorf("Failed: %v - Expected rmdCreated %v, got %v", tc.name, tc.rmdDSCreated, rmdCreated)
		}
		if nodeAgentCreated != tc.nodeAgentDSCreated {
			t.Errorf("Failed: %v - Expected node agent DS created %v, got %v", tc.name, tc.nodeAgentDSCreated, nodeAgentCreated)
		}

		//Check if rmdConfig status has updated
		rmdConfig := &intelv1alpha1.RmdConfig{}
		rmdConfigNamespacedName := types.NamespacedName{
			Namespace: defaultNamespace,
			Name:      "rmdconfig",
		}
		err = r.client.Get(context.TODO(), rmdConfigNamespacedName, rmdConfig)
		if err != nil {
			t.Errorf("Failed: %v - failed to get updated rmdconfig", tc.name)
		}
		if !reflect.DeepEqual(rmdConfig.Status.Nodes, tc.expectedRmdConfig.Status.Nodes) {
			t.Errorf("Failed: %v - expected rmdconfig status.nodes %v, got %v", tc.name, tc.expectedRmdConfig.Status.Nodes, rmdConfig.Status.Nodes)
		}
	}
}

func TestCreateDSIfNotPresent(t *testing.T) {
	tcases := []struct {
		name      string
		rmdConfig *intelv1alpha1.RmdConfig
		dsName    string
		path      string
		dsCreated bool
	}{
		{
			name: "test case 1 - create rmd-ds",
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
			},
			dsName:    rmdConst,
			path:      "../../../build/manifests/rmd-ds.yaml",
			dsCreated: true,
		},
		{
			name: "test case 2 - create node-agent",
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
			},
			dsName:    nodeAgentNameConst,
			path:      "../../../build/manifests/rmd-node-agent-ds.yaml",
			dsCreated: true,
		},
	}

	for _, tc := range tcases {
		// Create a Reconcile object with the scheme and fake client.
		r, err := createReconcileRmdConfigObject(tc.rmdConfig)
		if err != nil {
			t.Fatalf("error creating ReconcileNode object: (%v)", err)
		}
		err = r.createDaemonSetIfNotPresent(tc.rmdConfig, tc.path)
		if err != nil {
			t.Fatalf("r.createPodIfNotPresent returned error: (%v)", err)
		}

		dsCreated := true
		ds := &appsv1.DaemonSet{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: tc.dsName}, ds)
		if err != nil {
			dsCreated = false
		}
		if tc.dsCreated != dsCreated {
			t.Errorf("Failed: %v - Expected %v, got %v", tc.name, tc.dsCreated, dsCreated)
		}

	}
}

func TestCreateNodeStateIfNotPresent(t *testing.T) {
	tcases := []struct {
		name             string
		rmdConfig        *intelv1alpha1.RmdConfig
		nodeName         string
		nodeStateCreated bool
	}{
		{
			name: "create rmd node state",
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: rmdConfigConst,
				},
			},
			nodeName:         "example-node-1.com",
			nodeStateCreated: true,
		},
	}

	for _, tc := range tcases {
		// Create a ReconcileNode object with the scheme and fake client.
		r, err := createReconcileRmdConfigObject(tc.rmdConfig)
		if err != nil {
			t.Fatalf("error creating ReconcileRmdConfig object: (%v)", err)
		}

		err = r.createNodeStateIfNotPresent(tc.nodeName, tc.rmdConfig)
		if err != nil {
			t.Fatalf("r.createNodeStateIfNotPresent returned error: (%v)", err)
		}

		nodeStateCreated := true
		nodeState := &intelv1alpha1.RmdNodeState{}
		nodeStateName := fmt.Sprintf("%s%s", "rmd-node-state-", tc.nodeName)
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: nodeStateName}, nodeState)
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
		rmdConfig         *intelv1alpha1.RmdConfig
		rmdNode           *corev1.Node
		rmdPod            *corev1.Pod
		response          rmdCache.Infos
		expectedCacheWays int
		expectedError     bool
	}{
		{
			name: "test case 1",
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
			},
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
					Namespace: defaultNamespace,
					Labels:    map[string]string{"name": "rmd-pod"},
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
					NodeName: "example-node-1.com",
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
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
			},

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
					Namespace: defaultNamespace,
					Labels:    map[string]string{"name": "rmd-pod"},
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
					NodeName: "example-node-1.com",
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
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
			},
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
					Namespace: defaultNamespace,
					Labels:    map[string]string{"name": "rmd-pod"},
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
					NodeName: "example-node-1.com",
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
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
			},
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
					Namespace: defaultNamespace,
					Labels:    map[string]string{"name": "rmd-pod"},
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
					NodeName: "example-node-1.com",
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
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
			},
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
					Namespace: defaultNamespace,
					Labels:    map[string]string{"name": "rmd-pod"},
				},
				Spec: corev1.PodSpec{
					NodeName: "example-node-1.com",
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
		{
			name: "test case - no container port",
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
			},
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
					Namespace: defaultNamespace,
					Labels:    map[string]string{"name": "rmd-pod"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{},
						},
					},
					NodeName: "example-node-1.com",
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
		r, err := createReconcileRmdConfigObject(tc.rmdConfig)
		if err != nil {
			t.Fatalf("error creating ReconcileNode object: (%v)", err)
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
		err = r.client.Create(context.TODO(), tc.rmdNode)
		if err != nil {
			t.Fatalf("Failed to create node ")
		}

		expectedError := false
		err = r.updateNodeStatusCapacity(tc.rmdNode)
		if err != nil {
			expectedError = true
		}

		node := &corev1.Node{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: tc.rmdNode.GetObjectMeta().GetName(), Namespace: tc.rmdNode.GetObjectMeta().GetNamespace()}, node)
		if err != nil {
			t.Fatalf(fmt.Sprintf("%s%s", tc.name, " Could not get node"))
		}
		ways := node.Status.Capacity[l3Cache]
		expectedWays := resource.MustParse(strconv.Itoa(tc.expectedCacheWays))
		if ways != expectedWays || expectedError != tc.expectedError {
			t.Errorf("Failed %v, expected: %v and %v, got %v and %v", tc.name, expectedWays, tc.expectedError, ways, expectedError)
		}

		ts.Close()

	}

}
