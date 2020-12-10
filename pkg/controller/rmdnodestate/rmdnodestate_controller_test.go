package rmdnodestate

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

func createReconcileRmdNodeStateObject(rmdNodeState *intelv1alpha1.RmdNodeState) (*ReconcileRmdNodeState, error) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme

	// Add route Openshift scheme
	if err := apis.AddToScheme(s); err != nil {
		return nil, err
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{rmdNodeState}

	// Register operator types with the runtime scheme.
	s.AddKnownTypes(intelv1alpha1.SchemeGroupVersion)

	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)

	// Create a fake rmd client.
	rmdCl := rmd.NewDefaultOperatorRmdClient()

	// Create an empty state list
	rmdNodeData := &state.RmdNodeData{
		RmdNodeList: []string{},
	}

	// Create a ReconcileNode object with the scheme and fake client.
	r := &ReconcileRmdNodeState{client: cl, rmdClient: rmdCl, scheme: s, rmdNodeData: rmdNodeData}

	return r, nil

}

func TestNodeStateControllerReconcile(t *testing.T) {
	//TODO: Add more test cases.
	tcases := []struct {
		name                 string
		rmdNodeState         *intelv1alpha1.RmdNodeState
		response             []rmdtypes.RDTWorkLoad
		expectedRmdNodeState *intelv1alpha1.RmdNodeState
	}{
		{
			name: "test case 1",
			rmdNodeState: &intelv1alpha1.RmdNodeState{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-node-state-example-node-1",
					Namespace: "default",
				},
				Spec: intelv1alpha1.RmdNodeStateSpec{
					Node: "example-node-1",
				},
			},
			response: []rmdtypes.RDTWorkLoad{
				{
					ID:      "1",
					CoreIDs: []string{"0", "49"},
					Status:  "Successful",
					UUID:    "rmd-workload-a",
				},
			},
			expectedRmdNodeState: &intelv1alpha1.RmdNodeState{
				ObjectMeta: metav1.ObjectMeta{
					Name: "rmd-node-state-example-node-1",
				},
				Status: intelv1alpha1.RmdNodeStateStatus{
					Workloads: map[string]intelv1alpha1.WorkloadMap{
						"rmd-workload-a": {
							"ID":       "1",
							"Core IDs": "0,49",
							"Status":   "Successful",
						},
					},
				},
			},
		},
	}

	for _, tc := range tcases {
		// Create a ReconcileNode object with the scheme and fake client.
		r, err := createReconcileRmdNodeStateObject(tc.rmdNodeState)
		if err != nil {
			t.Fatalf("error creating ReconcileRmdNodeState object: (%v)", err)
		}

		nodeName := tc.rmdNodeState.GetObjectMeta().GetName()
		nodeNamespace := tc.rmdNodeState.GetObjectMeta().GetNamespace()
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      nodeName,
				Namespace: nodeNamespace,
			},
		}

		// create a listener with the desired port.
		address := "127.0.0.1:8080"
		l, err := net.Listen("tcp", address)
		if err != nil {
			t.Fatalf("Failed to create listener: %v", err)
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
		podName := fmt.Sprintf("%s%s", rmdPodNameConst, tc.rmdNodeState.Spec.Node)
		dummyRmdPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: nodeNamespace,
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
				PodIP: "127.0.0.1",
			},
		}
		err = r.client.Create(context.TODO(), dummyRmdPod)
		if err != nil {
			t.Fatalf("Failed to create dummy rmd pod")
		}

		res, err := r.Reconcile(req)
		if err != nil {
			t.Fatalf("reconcile: (%v)", err)
		}

		nodeState := &intelv1alpha1.RmdNodeState{}
		err = r.client.Get(context.TODO(), req.NamespacedName, nodeState)
		if err != nil {
			t.Fatalf("Failed to retrieve updated nodestate")
		}

		//Check the result of reconciliation to make sure it has the desired state.
		if res.Requeue {
			t.Error("reconcile unexpectedly requeued request")
		}

		if !reflect.DeepEqual(tc.expectedRmdNodeState.Status, nodeState.Status) {
			t.Errorf("Failed: %v - Expected %v, got %v", tc.name, tc.expectedRmdNodeState, nodeState)
		}

	}
}
