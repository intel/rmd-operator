package rmdworkload

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/intel/rmd-operator/pkg/apis"
	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	"github.com/intel/rmd-operator/pkg/rmd"
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

	// Create a ReconcileRmdWorkload object with the scheme and fake client.
	r := &ReconcileRmdWorkload{client: cl, rmdClient: rmdCl, scheme: s}

	return r, nil

}

func TestRmdWorkloadControllerReconcile(t *testing.T) {
	//TODO: Add more test cases.
	tcases := []struct {
		name                      string
		rmdWorkload               *intelv1alpha1.RmdWorkload
		rmdNodeStateList          *intelv1alpha1.RmdNodeStateList
		rmdPods                   *corev1.PodList
		getWorkloadsResponse      []rmdtypes.RDTWorkLoad
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
			rmdNodeStateList: &intelv1alpha1.RmdNodeStateList{
				Items: []intelv1alpha1.RmdNodeState{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-node-state-example-node.com",
							Namespace: "default",
						},
						Spec: intelv1alpha1.RmdNodeStateSpec{
							Node: "example-node.com",
						},
					},
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
			getWorkloadsResponse: []rmdtypes.RDTWorkLoad{
				{
					UUID: "rmd-workload-1",
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
			rmdNodeStateList: &intelv1alpha1.RmdNodeStateList{
				Items: []intelv1alpha1.RmdNodeState{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-node-state-example-node.com",
							Namespace: "default",
						},
						Spec: intelv1alpha1.RmdNodeStateSpec{
							Node: "example-node.com",
						},
					},
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
			getWorkloadsResponse: []rmdtypes.RDTWorkLoad{
				{
					UUID: "rmd-workload-1",
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
			rmdNodeStateList: &intelv1alpha1.RmdNodeStateList{
				Items: []intelv1alpha1.RmdNodeState{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-node-state-example-node.com",
							Namespace: "default",
						},
						Spec: intelv1alpha1.RmdNodeStateSpec{
							Node: "example-node.com",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-node-state-example-node-1.com",
							Namespace: "default",
						},
						Spec: intelv1alpha1.RmdNodeStateSpec{
							Node: "example-node-1.com",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-node-state-example-node-2.com",
							Namespace: "default",
						},
						Spec: intelv1alpha1.RmdNodeStateSpec{
							Node: "example-node-2.com",
						},
					},
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
			getWorkloadsResponse: []rmdtypes.RDTWorkLoad{
				{
					UUID:    "rmd-workload-1",
					ID:      "1",
					CosName: "0_49_guaranteed",
					Status:  "Successful",
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
			rmdNodeStateList: &intelv1alpha1.RmdNodeStateList{
				Items: []intelv1alpha1.RmdNodeState{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-node-state-example-node.com",
							Namespace: "default",
						},
						Spec: intelv1alpha1.RmdNodeStateSpec{
							Node: "example-node.com",
						},
					},
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

		rmdWorkloadName := tc.rmdWorkload.GetObjectMeta().GetName()
		rmdWorkloadNamespace := tc.rmdWorkload.GetObjectMeta().GetNamespace()
		rmdWorkloadNamespacedName := types.NamespacedName{
			Name:      rmdWorkloadName,
			Namespace: rmdWorkloadNamespace,
		}
		req := reconcile.Request{
			NamespacedName: rmdWorkloadNamespacedName,
		}

		// create a listener with the desired port.
		address := "127.0.0.1:8080"
		l, err := net.Listen("tcp", address)
		if err != nil {
			t.Fatalf("Failed to create listener")
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/v1/workloads/", (func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				b, err := json.Marshal(tc.getWorkloadsResponse)
				if err == nil {
					fmt.Fprintln(w, string(b[:]))
				}
			} else {
				if err == nil {
					fmt.Fprintln(w, "ok")
				}
			}
		}))

		ts := httptest.NewUnstartedServer(mux)

		ts.Listener.Close()
		ts.Listener = l

		// Start the server.
		ts.Start()

		for i := range tc.rmdNodeStateList.Items {
			err = r.client.Create(context.TODO(), &tc.rmdNodeStateList.Items[i])
			if err != nil {
				t.Fatalf("Failed to create rmd node states")
			}
		}
		for i := range tc.rmdPods.Items {
			err = r.client.Create(context.TODO(), &tc.rmdPods.Items[i])
			if err != nil {
				t.Fatalf("Failed to create dummy rmd pod")
			}
		}
		expectedError := false
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
		ts.Close()
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
					Namespace: "deafult",
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
					Namespace: "deafult",
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
					Namespace: "deafult",
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
					Namespace: "deafult",
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
					Namespace: "deafult",
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
		getWorkloadsResponse []rmdtypes.RDTWorkLoad
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
			getWorkloadsResponse: []rmdtypes.RDTWorkLoad{
				{
					UUID:    "rmd-workload-1",
					ID:      "1",
					CosName: "0_22_guaranteed",
					Status:  "Successful",
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
			getWorkloadsResponse: []rmdtypes.RDTWorkLoad{
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
		l, err := net.Listen("tcp", address)
		if err != nil {
			t.Fatalf("Failed to create listener")
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/v1/workloads/", (func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				b, err := json.Marshal(tc.getWorkloadsResponse)
				if err == nil {
					fmt.Fprintln(w, string(b[:]))
				}
			} else if r.Method == "POST" {
				if err == nil {
					fmt.Fprintln(w, "successful post")
				}
			}
		}))

		ts := httptest.NewUnstartedServer(mux)

		ts.Listener.Close()
		ts.Listener = l

		// Start the server.
		ts.Start()

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
		getWorkloadsResponse []rmdtypes.RDTWorkLoad
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
		l, err := net.Listen("tcp", address)
		if err != nil {
			t.Fatalf("Failed to create listener")
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/v1/workloads/", (func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "PATCH" {
				if err == nil {
					fmt.Fprintln(w, "OK")
				}
			}
		}))

		ts := httptest.NewUnstartedServer(mux)

		ts.Listener.Close()
		ts.Listener = l

		// Start the server.
		ts.Start()

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
