package rmdconfig

import (
	"context"

	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	rmd "github.com/intel/rmd-operator/pkg/rmd"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	rdtCatLabel           = "feature.node.kubernetes.io/cpu-rdt.RDTL3CA"
	rmdNodeLabelConst     = "rmd-node"
	defaultNamespace      = "default"
	rmdNodeStateNameConst = "rmd-node-state-"
	rmdPodNameConst       = "rmd-"
	nodeAgentNameConst    = "rmd-node-agent-"
	l3Cache               = "intel.com/l3_cache_ways"
)

var rmdPodPath = "/rmd-manifests/rmd-pod.yaml"
var nodeAgentPath = "/rmd-manifests/rmd-node-agent.yaml"

var log = logf.Log.WithName("controller_rmdconfig")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new RmdConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRmdConfig{client: mgr.GetClient(), rmdClient: rmd.NewClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rmdconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource RmdConfig
	err = c.Watch(&source.Kind{Type: &intelv1alpha1.RmdConfig{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner RmdConfig
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &intelv1alpha1.RmdConfig{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &intelv1alpha1.RmdNodeState{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &intelv1alpha1.RmdConfig{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRmdConfig implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRmdConfig{}

// ReconcileRmdConfig reconciles a RmdConfig object
type ReconcileRmdConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	rmdClient rmd.OperatorRmdClient
	scheme    *runtime.Scheme
}

// Reconcile reads that state of the cluster for a RmdConfig object and makes changes based on the state read
// and what is in the RmdConfig.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRmdConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling RmdConfig")

	// Fetch the RmdConfig instance
	instance := &intelv1alpha1.RmdConfig{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// List Nodes in cluster that already have RDT CAT label
	labelledNodeList := &corev1.NodeList{}
	listOption := client.MatchingLabels{
		rdtCatLabel: "true",
	}
	err = r.client.List(context.TODO(), labelledNodeList, client.MatchingLabels(listOption))
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("No Nodes found with RDT CAT label")
			return reconcile.Result{}, nil
		}
		reqLogger.Info("Failed to list Nodes")
		return reconcile.Result{}, err
	}

	for _, node := range labelledNodeList.Items {
		// Create RMD Daemonset if not present

		// Create Node Agent Daemonset if not present

		// Create RMD Node State if not present

		// Discover L3 cache ways on Node

		// Add new node state data to RmdNodeData object

	}

	return reconcile.Result{}, nil
}
