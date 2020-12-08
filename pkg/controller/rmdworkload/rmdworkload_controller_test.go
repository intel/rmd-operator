package rmdworkload

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/intel/rmd-operator/pkg/apis"
	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	"github.com/intel/rmd-operator/pkg/rmd"
	"github.com/intel/rmd-operator/pkg/state"
	rmdtypes "github.com/intel/rmd/modules/workload/types"
	corev1 "k8s.io/api/core/v1"
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
	"testing"
)

func createReconcileRmdWorkloadObject(rmdWorkload *intelv1alpha1.RmdWorkload) (*ReconcileRmdWorkload, error) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme

	// Add route Openshift scheme
	if err := apis.AddToScheme(s); err != nil {
		return nil, err
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{rmdWorkload}

	// Register operator types with the runtime scheme.
	s.AddKnownTypes(intelv1alpha1.SchemeGroupVersion)

	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)

	// Create a fake rmd client.
	rmdCl := rmd.NewDefaultOperatorRmdClient()

	rmdNodeData := &state.RmdNodeData{
		RmdNodeList: []string{},
	}

	// Create a ReconcileRmdWorkload object with the scheme and fake client.
	r := &ReconcileRmdWorkload{client: cl, rmdClient: rmdCl, scheme: s, rmdNodeData: rmdNodeData}

	return r, nil

}

func createListeners(address string, httpResponses map[string]([]rmdtypes.RDTWorkLoad)) (*httptest.Server, error) {
	var err error

	// create a listener with the desired port.
	newListener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Errorf("Failed to create listener")
	}

	//create muxes to handle get requests
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/workloads/", (func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			b, err := json.Marshal(httpResponses[address])
			if err == nil {
				fmt.Fprintln(w, string(b[:]))
			}
		}
	}))

	ts := httptest.NewUnstartedServer(mux)

	ts.Listener.Close()
	ts.Listener = newListener

	// Start the server.
	ts.Start()

	return ts, nil
}

func TestRmdWorkloadControllerReconcile(t *testing.T) {
	//TODO: Add more test cases.
	tcases := []struct {
		name                      string
		rmdWorkload               *intelv1alpha1.RmdWorkload
		rmdNodeData               []string
		rmdPods                   *corev1.PodList
		getWorkloadsResponse      map[string]([]rmdtypes.RDTWorkLoad)
		expectedRmdWorkloadStatus *intelv1alpha1.RmdWorkloadStatus
		expectedError             bool
	}{
		{
			name: "test case 1 - 1 RMD Node State, 1 RMD pod, no node in rmdWorkload spec",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-2",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Nodes: nil,
				},
			},
			rmdNodeData: []string{"example-node.com"},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string]([]rmdtypes.RDTWorkLoad){
				"127.0.0.1:8080": {
					{
						UUID: "rmd-workload-1",
					},
				},
			},
			expectedRmdWorkloadStatus: &intelv1alpha1.RmdWorkloadStatus{
				WorkloadStates: nil,
			},
			expectedError: false,
		},

		{
			name: "test case 2 - 1 RMD Node State, 1 RMD pod, single node in rmdWorkload spec, workload not present",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-2",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Nodes: []string{"example-node.com"},
				},
			},
			rmdNodeData: []string{"example-node.com"},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string]([]rmdtypes.RDTWorkLoad){
				"127.0.0.1:8080": {
					{
						UUID: "rmd-workload-1",
					},
				},
			},
			expectedRmdWorkloadStatus: &intelv1alpha1.RmdWorkloadStatus{
				WorkloadStates: map[string]intelv1alpha1.WorkloadState{
					"example-node.com": {
						Response: "Success: 200",
					},
				},
			},
			expectedError: false,
		},
		{
			name: "test case 3 - 3 RMD Node States, 3 RMD pods, workload present on all, 2 nodes in rmdWorkload spec",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Nodes: []string{"example-node.com", "example-node-2.com"},
				},
			},
			rmdNodeData: []string{"example-node.com", "example-node-1.com", "example-node-2.com"},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node-1.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.3",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node-2.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.2",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string]([]rmdtypes.RDTWorkLoad){
				"127.0.0.1:8080": {
					{
						UUID:    "rmd-workload-1",
						ID:      "1",
						CosName: "0_49_guaranteed",
						Status:  "Successful",
					},
				},
				"127.0.0.2:8080": {
					{
						UUID:    "rmd-workload-1",
						ID:      "1",
						CosName: "0_49_guaranteed",
						Status:  "Successful",
					},
				},
				"127.0.0.3:8080": {
					{
						UUID:    "rmd-workload-2",
						ID:      "2",
						CosName: "0_49_guaranteed",
						Status:  "Successful",
					},
				},
			},
			expectedRmdWorkloadStatus: &intelv1alpha1.RmdWorkloadStatus{
				WorkloadStates: map[string]intelv1alpha1.WorkloadState{
					"example-node.com": {
						Response: "Success: 200",
					},
					"example-node-2.com": {
						Response: "Success: 200",
					},
				},
			},
			expectedError: false,
		},

		{
			name: "test case 4 - 1 RMD Node State, 1 RMD pod, workload present",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Nodes: []string{"example-node.com-x"},
				},
			},
			rmdNodeData: []string{"example-node.com"},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
				},
			},
			expectedRmdWorkloadStatus: &intelv1alpha1.RmdWorkloadStatus{
				WorkloadStates: nil,
			},
			expectedError: true,
		},
	}

	for _, tc := range tcases {
		// Create a ReconcileNode object with the scheme and fake client.
		r, err := createReconcileRmdWorkloadObject(tc.rmdWorkload)
		if err != nil {
			t.Fatalf("error creating ReconcileRmdWorkload object: (%v)", err)
		}

		ts := make([]*httptest.Server, 0)
		for i := range tc.rmdPods.Items {
			//get address (i.e. IP and port number)
			podIP := tc.rmdPods.Items[i].Status.PodIPs[0].IP
			containerPort := tc.rmdPods.Items[i].Spec.Containers[0].Ports[0].ContainerPort
			address := fmt.Sprintf("%s:%v", podIP, containerPort)

			// Create listeners to manage http GET requests
			server, err := createListeners(address, tc.getWorkloadsResponse)
			if err != nil {
				t.Fatalf("error creating listeners: (%v)", err)
			}
			ts = append(ts, server)
		}

		rmdWorkloadName := tc.rmdWorkload.GetObjectMeta().GetName()
		rmdWorkloadNamespace := tc.rmdWorkload.GetObjectMeta().GetNamespace()
		rmdWorkloadNamespacedName := types.NamespacedName{
			Name:      rmdWorkloadName,
			Namespace: rmdWorkloadNamespace,
		}
		req := reconcile.Request{
			NamespacedName: rmdWorkloadNamespacedName,
		}

		for i := range tc.rmdPods.Items {
			err = r.client.Create(context.TODO(), &tc.rmdPods.Items[i])
			if err != nil {
				t.Fatalf("Failed to create dummy rmd pod")
			}
		}
		expectedError := false
		r.rmdNodeData.RmdNodeList = tc.rmdNodeData
		res, err := r.Reconcile(req)
		if err != nil {
			expectedError = true
		}

		//Check the result of reconciliation to make sure it has the desired state.
		if res.Requeue {
			t.Error("reconcile unexpectedly requeued request")
		}

		rmdWorkload := &intelv1alpha1.RmdWorkload{}
		err = r.client.Get(context.TODO(), rmdWorkloadNamespacedName, rmdWorkload)
		if err != nil {
			t.Fatalf("Failed to get workload after update")
		}

		if !reflect.DeepEqual(tc.expectedRmdWorkloadStatus, &rmdWorkload.Status) {
			t.Errorf("Failed: %v - Expected %v, got %v", tc.name, tc.expectedRmdWorkloadStatus, rmdWorkload.Status)
		}
		if tc.expectedError != expectedError {
			t.Errorf("Failed: %v - Expected %v, got %v", tc.name, tc.expectedError, expectedError)
		}
		for i := range tc.rmdPods.Items {
			//Close the listeners
			ts[i].Close()
		}
	}
}

func TestFindObseleteWorkloads(t *testing.T) {
	tcases := []struct {
		name                      string
		rmdNodeData               []string
		request                   reconcile.Request
		rmdWorkload               *intelv1alpha1.RmdWorkload
		rmdPods                   *corev1.PodList
		getWorkloadsResponse      map[string]([]rmdtypes.RDTWorkLoad)
		expectedObseleteWorkloads map[string]string
		expectedErr               bool
	}{
		{
			name:        "test case 1 - 1 obselete workload only",
			rmdNodeData: []string{"example-node.com"},
			request: reconcile.Request{
				types.NamespacedName{
					Namespace: "default",
					Name:      "rmd-workload-1",
				},
			},
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Nodes: []string{"example-node.com"},
				},
			},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string]([]rmdtypes.RDTWorkLoad){
				"127.0.0.1:8080": {
					{
						UUID: "rmd-workload-1",
						ID:   "1",
					},
				},
			},
			expectedObseleteWorkloads: map[string]string{
				"http://127.0.0.1:8080": "1",
			},
			expectedErr: false,
		},
		{
			name:        "test case 2 - 3 workloads, 1 obselete",
			rmdNodeData: []string{"example-node.com"},
			request: reconcile.Request{
				types.NamespacedName{
					Namespace: "default",
					Name:      "rmd-workload-1",
				},
			},
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Nodes: []string{"example-node.com"},
				},
			},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string]([]rmdtypes.RDTWorkLoad){
				"127.0.0.1:8080": {
					{
						UUID: "rmd-workload-1",
						ID:   "1",
					},
					{
						UUID: "rmd-workload-2",
						ID:   "2",
					},
					{
						UUID: "rmd-workload-3",
						ID:   "3",
					},
				},
			},
			expectedObseleteWorkloads: map[string]string{
				"http://127.0.0.1:8080": "1",
			},
			expectedErr: false,
		},
		{
			name:        "test case 3 - no obselete workload",
			rmdNodeData: []string{"example-node.com"},
			request: reconcile.Request{
				types.NamespacedName{
					Namespace: "",
					Name:      "",
				},
			},
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Nodes: []string{"example-node.com"},
				},
			},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string]([]rmdtypes.RDTWorkLoad){
				"127.0.0.1:8080": {
					{
						UUID: "rmd-workload-1",
						ID:   "1",
					},
					{
						UUID: "rmd-workload-2",
						ID:   "2",
					},
					{
						UUID: "rmd-workload-3",
						ID:   "3",
					},
				},
			},
			expectedObseleteWorkloads: map[string]string{},
			expectedErr:               false,
		},
		{
			name:        "test case 4 - multiple node states and pods, 1 obselete workload on both nodes",
			rmdNodeData: []string{"example-node.com", "example-node-2.com"},
			request: reconcile.Request{
				types.NamespacedName{
					Namespace: "default",
					Name:      "rmd-workload-1",
				},
			},
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Nodes: []string{"example-node.com"},
				},
			},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node-2.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8085,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.5",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string]([]rmdtypes.RDTWorkLoad){
				"127.0.0.1:8080": {
					{
						UUID: "rmd-workload-1",
						ID:   "1",
					},
					{
						UUID: "rmd-workload-2",
						ID:   "2",
					},
				},
				"127.0.0.5:8085": {
					{
						UUID: "rmd-workload-1",
						ID:   "3",
					},
					{
						UUID: "rmd-workload-4",
						ID:   "4",
					},
				},
			},
			expectedObseleteWorkloads: map[string]string{
				"http://127.0.0.1:8080": "1",
				"http://127.0.0.5:8085": "3",
			},
			expectedErr: false,
		},
		{
			name:        "test case 5 - multiple node states and pods, 1 obselete workload on one node only",
			rmdNodeData: []string{"example-node.com", "example-node-2.com"},
			request: reconcile.Request{
				types.NamespacedName{
					Namespace: "default",
					Name:      "rmd-workload-3",
				},
			},
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-3",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Nodes: []string{"example-node-2.com"},
				},
			},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node-2.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8085,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.5",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string]([]rmdtypes.RDTWorkLoad){
				"127.0.0.1:8080": {
					{
						UUID: "rmd-workload-1",
						ID:   "1",
					},
					{
						UUID: "rmd-workload-2",
						ID:   "2",
					},
				},
				"127.0.0.5:8085": {
					{
						UUID: "rmd-workload-3",
						ID:   "3",
					},
					{
						UUID: "rmd-workload-4",
						ID:   "4",
					},
				},
			},
			expectedObseleteWorkloads: map[string]string{
				"http://127.0.0.5:8085": "3",
			},
			expectedErr: false,
		},
	}

	for _, tc := range tcases {
		r, err := createReconcileRmdWorkloadObject(tc.rmdWorkload)
		if err != nil {
			t.Fatalf("error creating ReconcileRmdWorkload object: (%v)", err)
		}

		ts := make([]*httptest.Server, 0)
		for i := range tc.rmdPods.Items {
			//get address (i.e. IP and port number)
			podIP := tc.rmdPods.Items[i].Status.PodIPs[0].IP
			containerPort := tc.rmdPods.Items[i].Spec.Containers[0].Ports[0].ContainerPort
			address := fmt.Sprintf("%s:%v", podIP, containerPort)

			// Create listeners to manage http GET requests
			server, err := createListeners(address, tc.getWorkloadsResponse)
			if err != nil {
				t.Fatalf("error creating listeners: (%v)", err)
			}
			ts = append(ts, server)
		}

		for i := range tc.rmdPods.Items {
			err = r.client.Create(context.TODO(), &tc.rmdPods.Items[i])
			if err != nil {
				t.Fatalf("Failed to create dummy rmd pod")
			}
		}

		returnedErr := false
		r.rmdNodeData.RmdNodeList = tc.rmdNodeData
		obseleteWorkloads, err := r.findObseleteWorkloads(tc.request)
		if err != nil {
			returnedErr = true
		}
		if !reflect.DeepEqual(tc.expectedObseleteWorkloads, obseleteWorkloads) {
			t.Errorf("%v failed: Expected:  %v, Got:  %v\n", tc.name, tc.expectedObseleteWorkloads, obseleteWorkloads)
		}
		if tc.expectedErr != returnedErr {
			t.Errorf("%v failed: Expected error: %v, Error gotten: %v\n", tc.name, tc.expectedErr, returnedErr)
		}
		for i := range tc.rmdPods.Items {
			//Close the listeners
			ts[i].Close()
		}
	}
}

func TestFindTargetedNodes(t *testing.T) {
	tcases := []struct {
		name                 string
		request              reconcile.Request
		rmdNodeData          []string
		rmdWorkload          *intelv1alpha1.RmdWorkload
		rmdPods              *corev1.PodList
		getWorkloadsResponse map[string]([]rmdtypes.RDTWorkLoad)
		expectedWorkloads    []targetedNodeInfo
		expectedErr          bool
	}{
		{
			name: "test case 1 - workload to be added",
			request: reconcile.Request{
				types.NamespacedName{
					Namespace: "default",
					Name:      "rmd-workload-2",
				},
			},
			rmdNodeData: []string{"example-node.com"},
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-2",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Nodes: []string{"example-node.com"},
				},
			},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string][]rmdtypes.RDTWorkLoad{
				"127.0.0.1:8080": {
					{
						UUID: "rmd-workload-1",
						ID:   "1",
					},
				},
			},
			expectedWorkloads: []targetedNodeInfo{
				{
					nodeName:       "example-node.com",
					rmdAddress:     "http://127.0.0.1:8080",
					workloadExists: false,
				},
			},
			expectedErr: false,
		},
		{
			name: "test case 2 - workload to be updated",
			request: reconcile.Request{
				types.NamespacedName{
					Namespace: "default",
					Name:      "rmd-workload-2",
				},
			},
			rmdNodeData: []string{"example-node.com"},
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-2",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Policy: "gold",
					Nodes:  []string{"example-node.com"},
				},
			},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string][]rmdtypes.RDTWorkLoad{
				"127.0.0.1:8080": {
					{
						UUID: "rmd-workload-1",
						ID:   "1",
					},
					{
						UUID: "rmd-workload-2",
						ID:   "2",
					},
				},
			},
			expectedWorkloads: []targetedNodeInfo{
				{
					nodeName:       "example-node.com",
					rmdAddress:     "http://127.0.0.1:8080",
					workloadExists: true,
				},
			},
			expectedErr: false,
		},
		{
			name: "test case 3 - 2 nodestates, workload to be added on both nodes",
			request: reconcile.Request{
				types.NamespacedName{
					Namespace: "default",
					Name:      "rmd-workload-2",
				},
			},
			rmdNodeData: []string{"example-node.com", "example-node-2.com"},
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-2",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Nodes: []string{"example-node.com", "example-node-2.com"},
				},
			},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node-2.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8082,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.2",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string][]rmdtypes.RDTWorkLoad{
				"127.0.0.1:8080": {
					{
						UUID: "rmd-workload-1",
						ID:   "1",
					},
				},
				"127.0.0.2:8082": {
					{
						UUID: "rmd-workload-3",
						ID:   "3",
					},
				},
			},
			expectedWorkloads: []targetedNodeInfo{
				{
					nodeName:       "example-node.com",
					rmdAddress:     "http://127.0.0.1:8080",
					workloadExists: false,
				},
				{
					nodeName:       "example-node-2.com",
					rmdAddress:     "http://127.0.0.2:8082",
					workloadExists: false,
				},
			},
			expectedErr: false,
		},
		{
			name: "test case 4 - 3 nodestates, workload to be updated on 2 nodes",
			request: reconcile.Request{
				types.NamespacedName{
					Namespace: "default",
					Name:      "rmd-workload-2",
				},
			},
			rmdNodeData: []string{"example-node.com", "example-node-2.com", "example-node-3.com"},
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-2",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Policy: "gold",
					Nodes:  []string{"example-node.com", "example-node-3.com"},
				},
			},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node-2.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8082,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.2",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node-3.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8083,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.3",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string][]rmdtypes.RDTWorkLoad{
				"127.0.0.1:8080": {
					{
						UUID: "rmd-workload-1",
						ID:   "1",
					},
					{
						UUID: "rmd-workload-2",
						ID:   "2",
					},
				},
				"127.0.0.2:8082": {
					{
						UUID: "rmd-workload-3",
						ID:   "3",
					},
				},
				"127.0.0.3:8083": {
					{
						UUID: "rmd-workload-2",
						ID:   "2",
					},
					{
						UUID: "rmd-workload-4",
						ID:   "4",
					},
				},
			},
			expectedWorkloads: []targetedNodeInfo{
				{
					nodeName:       "example-node.com",
					rmdAddress:     "http://127.0.0.1:8080",
					workloadExists: true,
				},
				{
					nodeName:       "example-node-3.com",
					rmdAddress:     "http://127.0.0.3:8083",
					workloadExists: true,
				},
			},
			expectedErr: false,
		},
	}

	for _, tc := range tcases {
		r, err := createReconcileRmdWorkloadObject(tc.rmdWorkload)
		if err != nil {
			t.Fatalf("error creating ReconcileRmdWorkload object: (%v)", err)
		}

		ts := make([]*httptest.Server, 0)
		for i := range tc.rmdPods.Items {
			//get address (i.e. IP and port number)
			podIP := tc.rmdPods.Items[i].Status.PodIPs[0].IP
			containerPort := tc.rmdPods.Items[i].Spec.Containers[0].Ports[0].ContainerPort
			address := fmt.Sprintf("%s:%v", podIP, containerPort)

			// Create listeners to manage http GET requests
			server, err := createListeners(address, tc.getWorkloadsResponse)
			if err != nil {
				t.Fatalf("error creating listeners: (%v)", err)
			}
			ts = append(ts, server)
		}

		//make dummy pods
		for i := range tc.rmdPods.Items {
			err = r.client.Create(context.TODO(), &tc.rmdPods.Items[i])
			if err != nil {
				t.Fatalf("Failed to create dummy rmd pod")
			}
		}

		returnedErr := false
		r.rmdNodeData.RmdNodeList = tc.rmdNodeData
		returnedWorkloads, err := r.findTargetedNodes(tc.request, tc.rmdWorkload)
		if err != nil {
			returnedErr = true
		}

		if !reflect.DeepEqual(tc.expectedWorkloads, returnedWorkloads) {
			t.Errorf("%v failed: Expected:  %v, Got:  %v\n", tc.name, tc.expectedWorkloads, returnedWorkloads)
		}
		if tc.expectedErr != returnedErr {
			t.Errorf("%v failed: Expected error: %v, Error gotten: %v\n", tc.name, tc.expectedErr, returnedErr)
		}
		for i := range tc.rmdPods.Items {
			//Close the listeners
			ts[i].Close()
		}
	}
}

func TestFindRemovedNodes(t *testing.T) {
	tcases := []struct {
		name                 string
		request              reconcile.Request
		rmdNodeData          []string
		rmdWorkload          *intelv1alpha1.RmdWorkload
		rmdPods              *corev1.PodList
		getWorkloadsResponse map[string]([]rmdtypes.RDTWorkLoad)
		expectedNodes        map[string]string
		expectedError        bool
	}{
		{
			name: "test case 1 - workload to be deleted from 1 node",
			request: reconcile.Request{
				types.NamespacedName{
					Namespace: "default",
					Name:      "rmd-workload-2",
				},
			},
			rmdNodeData: []string{"example-node.com", "example-node-2.com"},
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-2",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Nodes: []string{"example-node.com"},
				},
			},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node-2.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8082,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.2",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string]([]rmdtypes.RDTWorkLoad){
				"127.0.0.1:8080": {
					{
						UUID: "rmd-workload-1",
						ID:   "1",
					},
				},
				"127.0.0.2:8082": {
					{
						UUID: "rmd-workload-2",
						ID:   "2",
					},
				},
			},
			expectedNodes: map[string]string{
				"http://127.0.0.2:8082": "2",
			},
			expectedError: false,
		},
		{
			name: "test case 2 - workload to be deleted from 2 nodes",
			request: reconcile.Request{
				types.NamespacedName{
					Namespace: "default",
					Name:      "rmd-workload-2",
				},
			},
			rmdNodeData: []string{"example-node.com", "example-node-2.com", "example-node-3.com"},
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-2",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Nodes: []string{"example-node-2.com"},
				},
			},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node-2.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8082,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.2",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node-3.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8083,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.3",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string]([]rmdtypes.RDTWorkLoad){
				"127.0.0.1:8080": {
					{
						UUID: "rmd-workload-1",
						ID:   "1",
					},
					{
						UUID: "rmd-workload-2",
						ID:   "2",
					},
				},
				"127.0.0.2:8082": {
					{
						UUID: "rmd-workload-3",
						ID:   "3",
					},
				},
				"127.0.0.3:8083": {
					{
						UUID: "rmd-workload-2",
						ID:   "4",
					},
					{
						UUID: "rmd-workload-5",
						ID:   "5",
					},
				},
			},
			expectedNodes: map[string]string{
				"http://127.0.0.1:8080": "2",
				"http://127.0.0.3:8083": "4",
			},
			expectedError: false,
		},
		{
			name: "test case 3 - no workload to be deleted",
			request: reconcile.Request{
				types.NamespacedName{
					Namespace: "default",
					Name:      "rmd-workload-2",
				},
			},
			rmdNodeData: []string{"example-node.com", "example-node-2.com", "example-node-3.com"},
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-2",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Nodes: []string{"example-node.com", "example-node-2.com", "example-node-3.com"},
				},
			},
			rmdPods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.1",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node-2.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8082,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.2",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-example-node-3.com",
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8083,
										},
									},
								},
							},
						},
						Status: corev1.PodStatus{
							PodIPs: []corev1.PodIP{
								{
									IP: "127.0.0.3",
								},
							},
						},
					},
				},
			},
			getWorkloadsResponse: map[string]([]rmdtypes.RDTWorkLoad){
				"127.0.0.1:8080": {
					{
						UUID: "rmd-workload-1",
						ID:   "1",
					},
					{
						UUID: "rmd-workload-2",
						ID:   "2",
					},
				},
				"127.0.0.2:8082": {
					{
						UUID: "rmd-workload-2",
						ID:   "3",
					},
					{
						UUID: "rmd-workload-3",
						ID:   "4",
					},
				},
				"127.0.0.3:8083": {
					{
						UUID: "rmd-workload-2",
						ID:   "5",
					},
					{
						UUID: "rmd-workload-5",
						ID:   "6",
					},
				},
			},
			expectedNodes: map[string]string{},
			expectedError: false,
		},
	}

	for _, tc := range tcases {
		r, err := createReconcileRmdWorkloadObject(tc.rmdWorkload)
		if err != nil {
			t.Fatalf("error creating ReconcileRmdWorkload object: (%v)", err)
		}

		ts := make([]*httptest.Server, 0)
		for i := range tc.rmdPods.Items {
			//get address (i.e. IP and port number)
			podIP := tc.rmdPods.Items[i].Status.PodIPs[0].IP
			containerPort := tc.rmdPods.Items[i].Spec.Containers[0].Ports[0].ContainerPort
			address := fmt.Sprintf("%s:%v", podIP, containerPort)

			// Create listeners to manage http GET requests
			server, err := createListeners(address, tc.getWorkloadsResponse)
			if err != nil {
				t.Fatalf("error creating listeners: (%v)", err)
			}
			ts = append(ts, server)
		}

		for i := range tc.rmdPods.Items {
			err = r.client.Create(context.TODO(), &tc.rmdPods.Items[i])
			if err != nil {
				t.Fatalf("Failed to create dummy rmd pod")
			}
		}

		returnedError := false
		r.rmdNodeData.RmdNodeList = tc.rmdNodeData
		removedNodes, err := r.findRemovedNodes(tc.request, tc.rmdWorkload)
		if err != nil {
			returnedError = true
		}
		if !reflect.DeepEqual(tc.expectedNodes, removedNodes) {
			t.Errorf("%v failed: Expected:  %v, Got:  %v\n", tc.name, tc.expectedNodes, removedNodes)
		}
		if tc.expectedError != returnedError {
			t.Errorf("%v failed: Expected error: %v, Error gotten: %v\n", tc.name, tc.expectedError, returnedError)
		}
		for i := range tc.rmdPods.Items {
			//Close the listeners
			ts[i].Close()
		}
	}
}

func TestGetPodAddress(t *testing.T) {
	tcases := []struct {
		name            string
		nodeName        string
		namespace       string
		rmdWorkload     *intelv1alpha1.RmdWorkload
		rmdPod          *corev1.Pod
		expectedAddress string
		expectedError   bool
	}{
		{
			name:      "test case 1",
			nodeName:  "example-node.com",
			namespace: "default",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
			},
			rmdPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8081,
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					PodIP: "127.0.0.1",
				},
			},
			expectedAddress: "http://127.0.0.1:8081",
			expectedError:   false,
		},
		{
			name:      "test case 2",
			nodeName:  "example-node.com",
			namespace: "default",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
			},
			rmdPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8081,
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					PodIPs: []corev1.PodIP{
						{
							IP: "127.0.0.1",
						},
					},
				},
			},
			expectedAddress: "http://127.0.0.1:8081",
			expectedError:   false,
		},
		{
			name:      "test case 3",
			nodeName:  "example-node.com",
			namespace: "default",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
			},
			rmdPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node.com",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					PodIPs: []corev1.PodIP{
						{
							IP: "127.0.0.1",
						},
					},
				},
			},
			expectedAddress: "",
			expectedError:   true,
		},
		{
			name:      "test case 4",
			nodeName:  "example-node.com",
			namespace: "default",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
			},
			rmdPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8081,
								},
							},
						},
					},
				},
			},
			expectedAddress: "",
			expectedError:   true,
		},
		{
			name:      "test case 5",
			nodeName:  "wrong-nodename-example-node.com",
			namespace: "default",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
			},
			rmdPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8081,
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
			expectedAddress: "",
			expectedError:   true,
		},
	}

	for _, tc := range tcases {
		// Create a ReconcileNode object with the scheme and fake client.
		r, err := createReconcileRmdWorkloadObject(tc.rmdWorkload)
		if err != nil {
			t.Fatalf("error creating ReconcileRmdNodeState object: (%v)", err)
		}

		err = r.client.Create(context.TODO(), tc.rmdPod)
		if err != nil {
			t.Fatalf("Failed to create dummy rmd pod")
		}

		errorReturned := false
		address, err := r.getPodAddress(tc.nodeName, tc.namespace)
		if err != nil {
			errorReturned = true
		}

		if address != tc.expectedAddress || errorReturned != tc.expectedError {
			t.Errorf("Failed: %v - Expected %v and %v, got %v and %v", tc.name, tc.expectedAddress, tc.expectedError, address, errorReturned)
		}

	}
}

func TestAddWorkload(t *testing.T) {
	tcases := []struct {
		name                 string
		nodeName             string
		address              string
		rmdWorkload          *intelv1alpha1.RmdWorkload
		getWorkloadsResponse map[string]([]rmdtypes.RDTWorkLoad)
		expectedRmdWorkload  *intelv1alpha1.RmdWorkload
	}{
		{
			name:     "test case 1",
			nodeName: "example-node.com",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
			},
			address: "127.0.0.1:8080",
			getWorkloadsResponse: map[string]([]rmdtypes.RDTWorkLoad){
				"127.0.0.1:8080": {
					{
						UUID:    "rmd-workload-1",
						ID:      "1",
						CosName: "0_22_guaranteed",
						Status:  "Successful",
					},
				},
			},
			expectedRmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
				Status: intelv1alpha1.RmdWorkloadStatus{
					WorkloadStates: map[string]intelv1alpha1.WorkloadState{
						"example-node.com": {
							Response: "Success: 200",
							ID:       "1",
							CosName:  "0_22_guaranteed",
							Status:   "Successful",
						},
					},
				},
			},
		},
		{
			name:     "test case 2",
			nodeName: "example-node.com",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-2",
					Namespace: "default",
				},
			},
			address: "127.0.0.1:8080",
			getWorkloadsResponse: map[string]([]rmdtypes.RDTWorkLoad){
				"127.0.0.1:8080": {
					{
						UUID:    "rmd-workload-1",
						ID:      "1",
						CosName: "0_49_guaranteed",
						Status:  "Successful",
					},
					{
						UUID:    "rmd-workload-2",
						ID:      "2",
						CosName: "1_50_guaranteed",
						Status:  "Successful",
					},
					{
						UUID:    "rmd-workload-3",
						ID:      "3",
						CosName: "2_52_guaranteed",
						Status:  "Successful",
					},
				},
			},
			expectedRmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-2",
					Namespace: "default",
				},
				Status: intelv1alpha1.RmdWorkloadStatus{
					WorkloadStates: map[string]intelv1alpha1.WorkloadState{
						"example-node.com": {
							Response: "Success: 200",
							ID:       "2",
							CosName:  "1_50_guaranteed",
							Status:   "Successful",
						},
					},
				},
			},
		},
		{
			name:     "test case 3",
			nodeName: "example-node.com",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
			},
			address: "127.0.0.x:xxxx",
			expectedRmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
				Status: intelv1alpha1.RmdWorkloadStatus{
					WorkloadStates: map[string]intelv1alpha1.WorkloadState{
						"example-node.com": {
							Response: "Failed to create new http post request",
						},
					},
				},
			},
		},
		{
			name:     "test case 4",
			nodeName: "example-node.com",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
			},
			address: "",
			expectedRmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
				Status: intelv1alpha1.RmdWorkloadStatus{
					WorkloadStates: map[string]intelv1alpha1.WorkloadState{
						"example-node.com": {
							Response: "Failed to set header for http post request",
						},
					},
				},
			},
		},
	}

	for _, tc := range tcases {
		// Create a ReconcileNode object with the scheme and fake client.
		r, err := createReconcileRmdWorkloadObject(tc.rmdWorkload)
		if err != nil {
			t.Fatalf("error creating ReconcileRmdNodeState object: (%v)", err)
		}

		// create a listener with the desired port.
		address := "127.0.0.1:8080"
		ts, err := createListeners(address, tc.getWorkloadsResponse)
		if err != nil {
			t.Fatalf("error creating Listener: (%v)", err)
		}

		err = r.addWorkload(fmt.Sprintf("%s%s", "http://", tc.address), tc.rmdWorkload, tc.nodeName)
		if err != nil {
			t.Errorf("Error from addWorkload: %v", err)
		}

		rmdWorkloadName := tc.rmdWorkload.GetObjectMeta().GetName()
		rmdWorkloadNamespace := tc.rmdWorkload.GetObjectMeta().GetNamespace()
		rmdWorkloadNamespacedName := types.NamespacedName{
			Name:      rmdWorkloadName,
			Namespace: rmdWorkloadNamespace,
		}
		rmdWorkload := &intelv1alpha1.RmdWorkload{}
		err = r.client.Get(context.TODO(), rmdWorkloadNamespacedName, rmdWorkload)
		if err != nil {
			t.Fatalf("Failed to get workload after update")
		}

		expectedWorkloadState := tc.expectedRmdWorkload.Status.WorkloadStates[tc.nodeName]
		actualWorkloadState := rmdWorkload.Status.WorkloadStates[tc.nodeName]

		if !reflect.DeepEqual(actualWorkloadState, expectedWorkloadState) {
			t.Errorf("Failed: %v - Expected %v, got %v", tc.name, expectedWorkloadState, actualWorkloadState)
		}

		ts.Close()

	}
}

func TestUpdateWorkload(t *testing.T) {
	tcases := []struct {
		name                 string
		nodeName             string
		address              string
		rmdWorkload          *intelv1alpha1.RmdWorkload
		getWorkloadsResponse map[string]([]rmdtypes.RDTWorkLoad)
		expectedRmdWorkload  *intelv1alpha1.RmdWorkload
	}{
		{
			name:     "test case 1",
			nodeName: "example-node.com",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
			},
			address: "127.0.0.1:8080",
			expectedRmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
				Status: intelv1alpha1.RmdWorkloadStatus{
					WorkloadStates: map[string]intelv1alpha1.WorkloadState{
						"example-node.com": {
							Response: "Success: 200",
						},
					},
				},
			},
		},
		{
			name:     "test case 2",
			nodeName: "example-node.com",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-2",
					Namespace: "default",
				},
			},
			address: "127.0.0.1:8080",
			expectedRmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-2",
					Namespace: "default",
				},
				Status: intelv1alpha1.RmdWorkloadStatus{
					WorkloadStates: map[string]intelv1alpha1.WorkloadState{
						"example-node.com": {
							Response: "Success: 200",
						},
					},
				},
			},
		},
		{
			name:     "test case 3",
			nodeName: "example-node.com",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
			},
			address: "127.0.0.x:xxxx",
			expectedRmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
				Status: intelv1alpha1.RmdWorkloadStatus{
					WorkloadStates: map[string]intelv1alpha1.WorkloadState{
						"example-node.com": {
							Response: "Failed to create new http patch request",
						},
					},
				},
			},
		},
		{
			name:     "test case 4",
			nodeName: "example-node.com",
			rmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
			},
			address: "",
			expectedRmdWorkload: &intelv1alpha1.RmdWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-workload-1",
					Namespace: "default",
				},
				Status: intelv1alpha1.RmdWorkloadStatus{
					WorkloadStates: map[string]intelv1alpha1.WorkloadState{
						"example-node.com": {
							Response: "Failed to set header for http patch request",
						},
					},
				},
			},
		},
	}

	for _, tc := range tcases {
		// Create a ReconcileNode object with the scheme and fake client.
		r, err := createReconcileRmdWorkloadObject(tc.rmdWorkload)
		if err != nil {
			t.Fatalf("error creating ReconcileRmdNodeState object: (%v)", err)
		}

		// create a listener with the desired port.
		address := "127.0.0.1:8080"
		ts, err := createListeners(address, tc.getWorkloadsResponse)
		if err != nil {
			t.Errorf("error creating Listener: (%v)", err)
		}

		err = r.updateWorkload(fmt.Sprintf("%s%s", "http://", tc.address), tc.rmdWorkload, tc.nodeName)
		if err != nil {
			t.Errorf("Error from addWorkload: %v", err)
		}

		rmdWorkloadName := tc.rmdWorkload.GetObjectMeta().GetName()
		rmdWorkloadNamespace := tc.rmdWorkload.GetObjectMeta().GetNamespace()
		rmdWorkloadNamespacedName := types.NamespacedName{
			Name:      rmdWorkloadName,
			Namespace: rmdWorkloadNamespace,
		}
		rmdWorkload := &intelv1alpha1.RmdWorkload{}
		err = r.client.Get(context.TODO(), rmdWorkloadNamespacedName, rmdWorkload)
		if err != nil {
			t.Fatalf("Failed to get workload after update")
		}

		expectedWorkloadState := tc.expectedRmdWorkload.Status.WorkloadStates[tc.nodeName]
		actualWorkloadState := rmdWorkload.Status.WorkloadStates[tc.nodeName]

		if !reflect.DeepEqual(actualWorkloadState, expectedWorkloadState) {
			t.Errorf("Failed: %v - Expected %v, got %v", tc.name, expectedWorkloadState, actualWorkloadState)
		}

		ts.Close()

	}
}
