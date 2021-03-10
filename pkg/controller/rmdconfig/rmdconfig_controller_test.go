package rmdconfig

import (
	"context"
	"fmt"
	"github.com/intel/rmd-operator/pkg/apis"
	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	"github.com/intel/rmd-operator/pkg/rmd"
	"github.com/intel/rmd-operator/pkg/state"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
					RmdImage:        "rmd:latest",
					DeployNodeAgent: true,
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
					RmdImage:        "rmd:latest",
					DeployNodeAgent: true,
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
					DeployNodeAgent: true,
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
					DeployNodeAgent: true,
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
		{
			name: "test case 5 - single node with nodeselector label, no node agent",
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
				Spec: intelv1alpha1.RmdConfigSpec{
					RmdImage:        "rmd:latest",
					DeployNodeAgent: false,
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
			nodeAgentDSCreated:   false,
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
			name: "test case 6 - single node with RDT L3 CAT label. No RmdNodeSelector defined - default set to RDT L3 CAT label, no node agent",
			rmdConfig: &intelv1alpha1.RmdConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rmdConfigConst,
					Namespace: defaultNamespace,
				},
				Spec: intelv1alpha1.RmdConfigSpec{
					RmdImage:        "rmd:latest",
					DeployNodeAgent: false,
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
			nodeAgentDSCreated:   false,
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
			name: "test case 7 - 3 nodes, 1 with all labels, 2 with some, no node agent",
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
					DeployNodeAgent: false,
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
			nodeAgentDSCreated:   false,
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
			name: "test case 8 - 3 nodes, 2 with RmdNodeSelector label, no node agent",
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
					DeployNodeAgent: false,
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
			nodeAgentDSCreated:   false,
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
