package rmdconfig

import (
	"context"
	"fmt"
	"io/ioutil"
	"strconv"

	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	rmd "github.com/intel/rmd-operator/pkg/rmd"
	"github.com/intel/rmd-operator/pkg/state"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
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
	rmdPodNameConst       = "rmd"
	nodeAgentNameConst    = "rmd-node-agent"
	l3Cache               = "intel.com/l3_cache_ways"
)

var rmdDaemonSetPath = "/rmd-manifests/rmd-ds.yaml"
var nodeAgentDaemonSetPath = "/rmd-manifests/rmd-node-agent-ds.yaml"

var log = logf.Log.WithName("controller_rmdconfig")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new RmdConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, rmdNodeData *state.RmdNodeData) error {
	return add(mgr, newReconciler(mgr, rmdNodeData))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, rmdNodeData *state.RmdNodeData) reconcile.Reconciler {
	return &ReconcileRmdConfig{client: mgr.GetClient(), rmdClient: rmd.NewClient(), scheme: mgr.GetScheme(), rmdNodeData: rmdNodeData}
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
	client      client.Client
	rmdClient   rmd.OperatorRmdClient
	scheme      *runtime.Scheme
	rmdNodeData *state.RmdNodeData
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
			reqLogger.Info("RmdConfig not found, return empty")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Info("Error reading RmdConfig, return empty")
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
	reqLogger.Info("NODES LISTED")

	for _, node := range labelledNodeList.Items {
		// Create RMD Daemonset if not present
		rmdNamespacedName := types.NamespacedName{
			Namespace: defaultNamespace,
			Name:      rmdPodNameConst,
		}
		err = r.createDaemonSetIfNotPresent(instance, rmdNamespacedName, rmdDaemonSetPath)
		if err != nil {
			reqLogger.Info("Failed to create RMD DS")
			return reconcile.Result{}, err
		}
		reqLogger.Info("RMD DS CREATED")

		// Create Node Agent Daemonset if not present
		nodeAgentNamespacedName := types.NamespacedName{
			Namespace: defaultNamespace,
			Name:      nodeAgentNameConst,
		}
		err = r.createDaemonSetIfNotPresent(instance, nodeAgentNamespacedName, nodeAgentDaemonSetPath)
		if err != nil {
			reqLogger.Info("Failed to create Node Agent DS")
			return reconcile.Result{}, err
		}
		reqLogger.Info("NODE AGENT DS CREATED")

		// Create RMD Node State if not present
		err = r.createNodeStateIfNotPresent(&node, instance)
		if err != nil {
			reqLogger.Info("Failed to create Node State")
			return reconcile.Result{}, err
		}
		reqLogger.Info("NODE STATE CREATED")
		// Discover L3 cache ways on Node
		err = r.updateNodeStatusCapacity(&node, rmdNamespacedName)
		if err != nil {
			reqLogger.Info("Failed to update cache ways")
			return reconcile.Result{}, err
		}
		reqLogger.Info("NODE STATUS CAPACITY UPDATED")
		// Add new node state data to RmdNodeData object
		r.rmdNodeData.UpdateRmdNodeData(node.Name)
		reqLogger.Info("RMD NODE DATA CREATED")

	}
	reqLogger.Info("RMD Config reconciled SUCCESSFULLY")
	return reconcile.Result{}, nil
}

func (r *ReconcileRmdConfig) createDaemonSetIfNotPresent(rmdConfig *intelv1alpha1.RmdConfig, namespacedName types.NamespacedName, path string) error {
	logger := log.WithName("createDaemonSetIfNotPresent")
	daemonSet := &appsv1.DaemonSet{}
	err := r.client.Get(context.TODO(), namespacedName, daemonSet)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create DaemonSet
			daemonSet, err = newDaemonSet(path, namespacedName)
			if err != nil {
				logger.Error(err, "Failed to build daemonSet from manifest")
				return err
			}
			if err := controllerutil.SetControllerReference(rmdConfig, daemonSet, r.scheme); err != nil {
				logger.Error(err, "unable to set owner reference on new daemonSet")
				return err
			}
			err = r.client.Create(context.TODO(), daemonSet)
			if err != nil {
				logger.Error(err, "Failed to create daemonSet")
				return err
			}
			logger.Info("New daemonSet created.")
		}
	}
	return nil
}

func (r *ReconcileRmdConfig) createNodeStateIfNotPresent(node *corev1.Node, rmdConfig *intelv1alpha1.RmdConfig) error {
	logger := log.WithName("createNodeStateIfNotPresent")
	rmdNodeState := &intelv1alpha1.RmdNodeState{}
	nodeName := string(node.GetObjectMeta().GetName())
	rmdNodeStateName := fmt.Sprintf("%s%s", rmdNodeStateNameConst, nodeName)
	namespacedName := types.NamespacedName{
		Namespace: defaultNamespace,
		Name:      rmdNodeStateName,
	}
	err := r.client.Get(context.TODO(), namespacedName, rmdNodeState)
	if err != nil {
		if errors.IsNotFound(err) {
			// RmdNodeState not found, create it
			rmdNodeState.SetName(namespacedName.Name)
			rmdNodeState.SetNamespace(namespacedName.Namespace)
			rmdNodeState.Spec = intelv1alpha1.RmdNodeStateSpec{
				Node:    node.GetObjectMeta().GetName(),
				NodeUID: string(node.GetObjectMeta().GetUID()),
			}
			workloads := make(map[string]intelv1alpha1.WorkloadMap)
			rmdNodeState.Status.Workloads = workloads
			if err := controllerutil.SetControllerReference(rmdConfig, rmdNodeState, r.scheme); err != nil {
				logger.Error(err, "unable to set owner reference on new service")
				return err
			}

			err = r.client.Create(context.TODO(), rmdNodeState)
			if err != nil {
				logger.Error(err, "Failed to create RmdNodeState")
				return err
			}
			logger.Info("RmdNodeState created.")
		}
	}

	return nil
}

func (r *ReconcileRmdConfig) updateNodeStatusCapacity(rmdNode *corev1.Node, rmdPodNamespacedName types.NamespacedName) error {
	logger := log.WithName("updateNodeStatusCapacity")

	pods := &corev1.PodList{}
	err := r.client.List(context.TODO(), pods, client.MatchingLabels(client.MatchingLabels{"name": "rmd-pod"}))
	if err != nil {
		logger.Info("Failed to list Pods")
		return err
	}
	podName := ""
	for _, pod := range pods.Items {
		for _, address := range rmdNode.Status.Addresses {
			if address.Address == pod.Status.HostIP {
				podName = pod.GetObjectMeta().GetName()
				break
			}
		}
	}
	rmdPodNamespacedName.Name = podName

	rmdPod := &corev1.Pod{}
	err = r.client.Get(context.TODO(), rmdPodNamespacedName, rmdPod)
	if err != nil {
		return err
	}
	// Query RMD for available cache ways, update node extended resources accordingly.
	err = errors.NewServiceUnavailable("rmdPod unavailable, requeuing")
	if len(rmdPod.Spec.Containers) == 0 {
		return err
	}
	if len(rmdPod.Spec.Containers[0].Ports) == 0 {
		return err
	}

	addressPrefix := r.rmdClient.GetAddressPrefix()
	address := fmt.Sprintf("%s%s%s%d", addressPrefix, rmdPod.Status.PodIP, ":", rmdPod.Spec.Containers[0].Ports[0].ContainerPort)

	availableCacheWays, err := r.rmdClient.GetAvailableCacheWays(address)
	if err != nil {
		// Cannot access l3 cache extended resources so set to zero.
		for extendedResource := range rmdNode.Status.Capacity {
			if extendedResource.String() == l3Cache {
				rmdNode.Status.Capacity[extendedResource] = resource.MustParse("0")
			}
		}

		err = r.client.Status().Update(context.TODO(), rmdNode)
		if err != nil {
			logger.Error(err, "failed to update the node with extended resource")
			return err
		}

		return nil
	}
	// If l3_cache_ways extended resource does not exist on the node or is zero,
	// update the node status capacity.
	if _, ok := rmdNode.Status.Capacity[corev1.ResourceName(l3Cache)]; !ok || rmdNode.Status.Capacity[corev1.ResourceName(l3Cache)] == resource.MustParse("0") {
		rmdNode.Status.Capacity[corev1.ResourceName(l3Cache)] = resource.MustParse(strconv.Itoa(int(availableCacheWays)))
		err = r.client.Status().Update(context.TODO(), rmdNode)
		if err != nil {
			logger.Error(err, "failed to update the node with extended resource")
			return err
		}
	}

	return nil
}

func newDaemonSet(path string, namespacedName types.NamespacedName) (*appsv1.DaemonSet, error) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error(err, "Error reading DaemonSet manifest")
		return nil, err
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(yamlFile, nil, nil)
	if err != nil {
		log.Error(err, "Error decoding DaemonSet manifest")
		return nil, err
	}

	rmdDaemonSet := obj.(*appsv1.DaemonSet)

	rmdDaemonSet.GetObjectMeta().SetName(namespacedName.Name)
	rmdDaemonSet.GetObjectMeta().SetNamespace(namespacedName.Namespace)
	//nodeLabel := make(map[string]string)
	//nodeLabel[rmdNodeLabelConst] = nodeUID
	//rmdDaemonSet.Spec.Template.NodeSelector = nodeLabel

	return rmdDaemonSet, nil
}
