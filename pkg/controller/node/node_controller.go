package node

import (
	"context"
	"fmt"
	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	rmd "github.com/intel/rmd-operator/pkg/rmd"
	"github.com/intel/rmd-operator/pkg/state"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"strconv"

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

var log = logf.Log.WithName("controller_node")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Node Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, rmdNodeData *state.RmdNodeData) error {
	return add(mgr, newReconciler(mgr, rmdNodeData))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, rmdNodeData *state.RmdNodeData) reconcile.Reconciler {
	return &ReconcileNode{client: mgr.GetClient(), rmdClient: rmd.NewClient(), scheme: mgr.GetScheme(), rmdNodeData: rmdNodeData}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("node-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Node
	err = c.Watch(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Node
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &corev1.Node{},
	})
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &intelv1alpha1.RmdNodeState{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &corev1.Node{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileNode implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNode{}

// ReconcileNode reconciles a Node object
type ReconcileNode struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client      client.Client
	rmdClient   rmd.OperatorRmdClient
	scheme      *runtime.Scheme
	rmdNodeData *state.RmdNodeData
}

// Reconcile reads that state of the cluster for a Node object and makes changes based on the state read
// and what is in the Node.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNode) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Node")

	rmdNode := &corev1.Node{}
	err := r.client.Get(context.TODO(), request.NamespacedName, rmdNode)
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

	// Return if node does not have RDT-CAT label
	// TODO: Extend check to include other RMD plugins (P-State, QAT etc)
	existingNodeLabels := labels.Set(rmdNode.GetObjectMeta().GetLabels())
	if !existingNodeLabels.Has(rdtCatLabel) {
		return reconcile.Result{}, nil
	}

	if existingNodeLabels.Get(rdtCatLabel) != "true" {
		return reconcile.Result{}, nil
	}

	// Add label "rmd-node":<node-UID> to rmdNode if not already present
	err = r.addNodeLabelIfNotPresent(rmdNode)
	if err != nil {
		reqLogger.Error(err, "Failed to update node")
		return reconcile.Result{}, err

	}

	// If RmdPod does not exist for RMD Node, create it.
	nodeName := string(rmdNode.GetObjectMeta().GetName())
	rmdPodName := fmt.Sprintf("%s%s", rmdPodNameConst, nodeName)
	rmdPodNamespacedName := types.NamespacedName{
		Namespace: defaultNamespace,
		Name:      rmdPodName,
	}
	err = r.createPodIfNotPresent(rmdNode, rmdPodNamespacedName, rmdPodPath)
	if err != nil {
		return reconcile.Result{}, err
	}

	// If RmdNodeAgent does not exist for RMD Node, create it.
	rmdNodeAgentName := fmt.Sprintf("%s%s", nodeAgentNameConst, nodeName)
	rmdNodeAgentNamespacedName := types.NamespacedName{
		Namespace: defaultNamespace,
		Name:      rmdNodeAgentName,
	}
	err = r.createPodIfNotPresent(rmdNode, rmdNodeAgentNamespacedName, nodeAgentPath)
	if err != nil {
		return reconcile.Result{}, err
	}

	// If RmdNodeState does not exist for RMD Node, create it.
	// RmdNodeState Name is "rmd-node-state-<node-UID>".
	rmdNodeStateName := fmt.Sprintf("%s%s", rmdNodeStateNameConst, nodeName)
	rmdNodeStateNamespacedName := types.NamespacedName{
		Namespace: defaultNamespace,
		Name:      rmdNodeStateName,
	}
	err = r.createNodeStateIfNotPresent(rmdNode, rmdNodeStateNamespacedName)
	if err != nil {
		return reconcile.Result{}, err
	}
	// Add new node state data to RmdNodeData object
	r.rmdNodeData.UpdateRmdNodeData(nodeName)

	// Update NodeStatus Capacity with l3 cache ways
	err = r.updateNodeStatusCapacity(rmdNode, rmdPodNamespacedName)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileNode) addNodeLabelIfNotPresent(rmdNode *corev1.Node) error {
	nodeUID := string(rmdNode.GetObjectMeta().GetUID())
	rmdNodeLabel := labels.Set{
		rmdNodeLabelConst: nodeUID,
	}

	existingNodeLabels := labels.Set(rmdNode.GetObjectMeta().GetLabels())
	if !existingNodeLabels.Has(rmdNodeLabelConst) {
		updatedLabels := labels.Merge(existingNodeLabels, rmdNodeLabel)
		rmdNode.GetObjectMeta().SetLabels(updatedLabels)
		err := r.client.Update(context.TODO(), rmdNode)
		if err != nil {
			return err
		}

	}

	return nil
}

func (r *ReconcileNode) createPodIfNotPresent(node *corev1.Node, namespacedName types.NamespacedName, path string) error {
	logger := log.WithName("createPodIfNotPresent")
	pod := &corev1.Pod{}
	err := r.client.Get(context.TODO(), namespacedName, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create RMD Pod for RMD Node.
			pod, err = newPod(path, string(node.GetObjectMeta().GetUID()), namespacedName)
			if err != nil {
				logger.Error(err, "Failed to build pod from manifest")
				return err
			}
			if err := controllerutil.SetControllerReference(node, pod, r.scheme); err != nil {
				logger.Error(err, "unable to set owner reference on new pod")
				return err
			}
			err = r.client.Create(context.TODO(), pod)
			if err != nil {
				logger.Error(err, "Failed to create pod")
				return err
			}
			logger.Info("New pod created.")
		}
	}
	return nil

}

func (r *ReconcileNode) createNodeStateIfNotPresent(node *corev1.Node, namespacedName types.NamespacedName) error {
	logger := log.WithName("createNodeStateIfNotPresent")
	rmdNodeState := &intelv1alpha1.RmdNodeState{}
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
			if err := controllerutil.SetControllerReference(node, rmdNodeState, r.scheme); err != nil {
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

func (r *ReconcileNode) updateNodeStatusCapacity(rmdNode *corev1.Node, rmdPodNamespacedName types.NamespacedName) error {
	logger := log.WithName("updateNodeStatusCapacity")
	rmdPod := &corev1.Pod{}
	err := r.client.Get(context.TODO(), rmdPodNamespacedName, rmdPod)
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

func newPod(path string, nodeUID string, namespacedName types.NamespacedName) (*corev1.Pod, error) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(yamlFile, nil, nil)
	if err != nil {
		return nil, err
	}

	rmdPod := obj.(*corev1.Pod)

	rmdPod.GetObjectMeta().SetName(namespacedName.Name)
	rmdPod.GetObjectMeta().SetNamespace(namespacedName.Namespace)
	nodeLabel := make(map[string]string)
	nodeLabel[rmdNodeLabelConst] = nodeUID
	rmdPod.Spec.NodeSelector = nodeLabel

	return rmdPod, nil

}
