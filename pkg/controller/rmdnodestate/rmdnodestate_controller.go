package rmdnodestate

import (
	"context"
	"fmt"
	"strings"
	"time"

	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	"github.com/intel/rmd-operator/pkg/rmd"
	"github.com/intel/rmd-operator/pkg/state"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	rmdPodNameConst = "rmd-"
)

var log = logf.Log.WithName("controller_rmdnodestate")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new RmdNodeState Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, rmdNodeData *state.RmdNodeData) error {
	return add(mgr, newReconciler(mgr, rmdNodeData))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, rmdNodeData *state.RmdNodeData) reconcile.Reconciler {
	return &ReconcileRmdNodeState{client: mgr.GetClient(), rmdClient: rmd.NewClient(), scheme: mgr.GetScheme(), rmdNodeData: rmdNodeData}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rmdnodestate-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource RmdNodeState
	err = c.Watch(&source.Kind{Type: &intelv1alpha1.RmdNodeState{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner RmdNodeState
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &intelv1alpha1.RmdNodeState{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRmdNodeState implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRmdNodeState{}

// ReconcileRmdNodeState reconciles a RmdNodeState object
type ReconcileRmdNodeState struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client      client.Client
	rmdClient   rmd.OperatorRmdClient
	scheme      *runtime.Scheme
	rmdNodeData *state.RmdNodeData
}

// Reconcile reads that state of the cluster for a RmdNodeState object and makes changes based on the state read
// and what is in the RmdNodeState.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRmdNodeState) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling RmdNodeState")

	// Fetch the RmdNodeState instance
	rmdNodeState := &intelv1alpha1.RmdNodeState{}
	err := r.client.Get(context.TODO(), request.NamespacedName, rmdNodeState)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			// Remove associated RmdNodeData entry
			nodeName := strings.ReplaceAll(request.Name, "rmd-node-state-", "")
			r.rmdNodeData.DeleteRmdNodeData(nodeName)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	rmdPodName := fmt.Sprintf("%s%s", rmdPodNameConst, rmdNodeState.Spec.Node)
	rmdPodNamespacedName := types.NamespacedName{
		Namespace: request.Namespace,
		Name:      rmdPodName,
	}
	rmdPod := &corev1.Pod{}
	err = r.client.Get(context.TODO(), rmdPodNamespacedName, rmdPod)
	if err != nil {
		reqLogger.Error(err, "Failed to get RMD pod")
		return reconcile.Result{}, err
	}

	if len(rmdPod.Spec.Containers) == 0 {
		return reconcile.Result{}, err
	}
	if len(rmdPod.Spec.Containers[0].Ports) == 0 {
		return reconcile.Result{}, err
	}
	addressPrefix := r.rmdClient.GetAddressPrefix()
	address := fmt.Sprintf("%s%s%s%d", addressPrefix, rmdPod.Status.PodIP, ":", rmdPod.Spec.Containers[0].Ports[0].ContainerPort)

	existingWorkloads, err := r.rmdClient.GetWorkloads(address)
	if err != nil {
		reqLogger.Info("Could not GET workloads.", "Error:", err)
	}

	workloadMap := make(map[string]intelv1alpha1.WorkloadMap)
	for _, existingWorkload := range existingWorkloads {
		workloadMap[existingWorkload.UUID], err = rmd.UpdateNodeStatusWorkload(existingWorkload)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	rmdNodeState.Status.Workloads = workloadMap

	err = r.client.Status().Update(context.TODO(), rmdNodeState)
	if err != nil {
		reqLogger.Error(err, "Failed to update RmdNodeState")
		return reconcile.Result{}, err
	}

	// Requeue every 5 seconds to keep RmdNodeState up to date with RMD instance.
	return reconcile.Result{RequeueAfter: time.Second * 5}, nil
}
