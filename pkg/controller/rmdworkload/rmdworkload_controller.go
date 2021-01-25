package rmdworkload

import (
	"context"
	"fmt"
	"strings"
	"time"

	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	rmd "github.com/intel/rmd-operator/pkg/rmd"
	"github.com/intel/rmd-operator/pkg/state"
	"github.com/intel/rmd-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
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
	defaultNamespace     = "default"
	rmdWorkloadNameConst = "-rmd-workload-"
	rmdPodNameConst      = "rmd-pod"
	rmdConfigConst       = "rmdconfig"
)

var log = logf.Log.WithName("controller_rmdworkload")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new RmdWorkload Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, rmdNodeData *state.RmdNodeData) error {
	return add(mgr, newReconciler(mgr, rmdNodeData))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, rmdNodeData *state.RmdNodeData) reconcile.Reconciler {
	return &ReconcileRmdWorkload{client: mgr.GetClient(), rmdClient: rmd.NewClient(), scheme: mgr.GetScheme(), rmdNodeData: rmdNodeData}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rmdworkload-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource RmdWorkload
	err = c.Watch(&source.Kind{Type: &intelv1alpha1.RmdWorkload{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner RmdWorkload
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &intelv1alpha1.RmdWorkload{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRmdWorkload implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRmdWorkload{}

// ReconcileRmdWorkload reconciles a RmdWorkload object
type ReconcileRmdWorkload struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client      client.Client
	rmdClient   rmd.OperatorRmdClient
	scheme      *runtime.Scheme
	rmdNodeData *state.RmdNodeData
}

//targetedNodeInfo is returned by r.findTargetedNodes()
type targetedNodeInfo struct {
	nodeName       string
	rmdAddress     string
	workloadExists bool
}

type removedNodeInfo struct {
	nodeName   string
	rmdAddress string
	workloadID string
}

// Reconcile reads that state of the cluster for a RmdWorkload object and makes changes based on the state read
// and what is in the RmdWorkload.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRmdWorkload) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling RmdWorkload")

	// In the event that an RmdWorkload has been deleted, we must also delete the corresponding
	// workload from the RMD instance. As we now only have the name of the workload being
	// reconciled after deletion, we must list all RmdNodeStates to discover which nodes
	// the deleted RmdWorkload currently exist on and remove accordingly.

	// Fetch the RmdWorkload instance
	rmdWorkload := &intelv1alpha1.RmdWorkload{}
	err := r.client.Get(context.TODO(), request.NamespacedName, rmdWorkload)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found (i.e. deleted)
			obseleteWorkloads, err := r.findObseleteWorkloads(request)
			if err != nil {
				return reconcile.Result{}, err
			}
			for address, workloadID := range obseleteWorkloads {
				err = r.rmdClient.DeleteWorkload(address, workloadID)
				if err != nil {
					reqLogger.Error(err, "Failed to delete workload from RMD")
					return reconcile.Result{}, err
				}
			}
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Discover all RMD instances that the reconciled RmdWorkload is targeting.
	// Add or Update those instances with the reconciled RmdWorkload accordingly
	targetedNodes, err := r.findTargetedNodes(request, rmdWorkload)
	if err != nil {
		return reconcile.Result{}, err
	}

	for _, targetedNode := range targetedNodes {
		if targetedNode.workloadExists == false {
			reqLogger.Info("Workload not found on RMD instance, create.")
			err := r.addWorkload(targetedNode.rmdAddress, rmdWorkload, targetedNode.nodeName)
			if err != nil {
				return reconcile.Result{}, err
			}
		} else {
			reqLogger.Info("Workload found on RMD instance, update.")
			err := r.updateWorkload(targetedNode.rmdAddress, rmdWorkload, targetedNode.nodeName)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// Perform final check to find workloads that need to be removed due to a change
	// in the reconciled RmdWorkload. Nodes may have been removed from the reconciled
	// RmdWorkload.Spec. In which case the reconciled workload
	// needs to be deleted from the Node's RMD.
	removedNodes, err := r.findRemovedNodes(request, rmdWorkload)
	if err != nil {
		reqLogger.Error(err, "Failed to find workloads to delete")
		return reconcile.Result{}, err
	}

	err = r.removeWorkload(rmdWorkload, removedNodes)
	if err != nil {
		reqLogger.Error(err, "Failed to remove workload")
		return reconcile.Result{}, err
	}

	// Requeue after 60 seconds. This is to account a change in node labels.
	// Should a node no longer possess the feature label(s) specified in the
	// RmdWorkload NodeSelector, the workload will be removed.
	return reconcile.Result{RequeueAfter: time.Second * 60}, nil
}

func (r *ReconcileRmdWorkload) findObseleteWorkloads(request reconcile.Request) (map[string]string, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	obseleteWorkloads := make(map[string]string)

	for _, nodeName := range r.rmdNodeData.RmdNodeList {
		address, err := r.getPodAddress(nodeName)
		if err != nil {
			return nil, err
		}

		activeWorkloads, err := r.rmdClient.GetWorkloads(address)
		if err != nil {
			reqLogger.Info("Could not GET workloads.", "Error:", err)
			return nil, err
		}

		workload := rmd.FindWorkloadByName(activeWorkloads, request.NamespacedName.Name)
		if workload.UUID == "" {
			reqLogger.Info("Workload not found on RMD instance")
			continue
		}
		obseleteWorkloads[address] = workload.ID
	}
	return obseleteWorkloads, nil
}

func (r *ReconcileRmdWorkload) getTargetedNode(nodeName, rmdWorkloadName string) (targetedNodeInfo, error) {
	targetedNode := targetedNodeInfo{}
	address, err := r.getPodAddress(nodeName)
	if err != nil {
		return targetedNode, err
	}

	activeWorkloads, err := r.rmdClient.GetWorkloads(address)
	if err != nil {
		return targetedNode, err
	}

	workloadExists := true
	workload := rmd.FindWorkloadByName(activeWorkloads, rmdWorkloadName)
	if workload.UUID == "" {
		workloadExists = false
	}
	targetedNode = targetedNodeInfo{nodeName, address, workloadExists}

	return targetedNode, nil
}

// findTargetedNodes returns information on each node that contains the RmdWorkload under reconciliation
func (r *ReconcileRmdWorkload) findTargetedNodes(request reconcile.Request, rmdWorkload *intelv1alpha1.RmdWorkload) ([]targetedNodeInfo, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	if len(rmdWorkload.Spec.NodeSelector) == 0 {
		targetedNodes := make([]targetedNodeInfo, 0)
		// Loop through nodes listed in RmdWorkload Spec.
		for _, nodeName := range rmdWorkload.Spec.Nodes {
			targetedNode, err := r.getTargetedNode(nodeName, rmdWorkload.GetObjectMeta().GetName())
			if err != nil {
				reqLogger.Error(err, "Failed to get targeted node. Has node name been entered correctly in RmdWorkload spec?")
				continue
				//	return nil, err
			}
			targetedNodes = append(targetedNodes, targetedNode)
		}
		return targetedNodes, nil

	}

	// If nodeSelector has been specified in RmdWorkload, only consider nodes with matching labels.
	targetedNodes := make([]targetedNodeInfo, 0)

	// Get rmdConfig object and merge RmdWorkload nodeSelector labels with RmdConfig
	// RmdNodeSelector labels. This is to ensure the request is not sent to a node on
	// which RMD does not exist.
	rmdConfig := &intelv1alpha1.RmdConfig{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: rmdConfigConst, Namespace: defaultNamespace}, rmdConfig)
	if err != nil {
		reqLogger.Error(err, "Failed to get RmdConfig object")
		return nil, err
	}

	matchingLabels := labels.Merge(rmdConfig.Spec.RmdNodeSelector, rmdWorkload.Spec.NodeSelector)
	labelsMap := make(map[string]string)
	labelsMap = matchingLabels
	nodeList := &corev1.NodeList{}
	err = r.client.List(context.TODO(), nodeList, client.MatchingLabels(labelsMap))
	if err != nil {
		reqLogger.Error(err, "Failed to list nodes according to RmdWorkload nodeSelector")
		return nil, err
	}

	for _, node := range nodeList.Items {
		targetedNode, err := r.getTargetedNode(node.GetObjectMeta().GetName(), rmdWorkload.GetObjectMeta().GetName())
		if err != nil {
			reqLogger.Error(err, "Failed to get targeted node")
			return nil, err
		}
		targetedNodes = append(targetedNodes, targetedNode)
	}

	return targetedNodes, nil
}

// findRemovedNodes finds Nodes that have the reconciled workload actively running, but those Nodes have been
// removed from the RmdWorkload spec. Such instances are returned as a map of address (of RMD Pod) to workload
// ID so that the workload can be deleted from RMD.
func (r *ReconcileRmdWorkload) findRemovedNodes(request reconcile.Request, rmdWorkload *intelv1alpha1.RmdWorkload) ([]removedNodeInfo, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	removedNodes := make([]removedNodeInfo, 0)
	rmdWorkloadName := rmdWorkload.GetObjectMeta().GetName()

	for _, nodeName := range r.rmdNodeData.RmdNodeList {
		address, err := r.getPodAddress(nodeName)
		if err != nil {
			reqLogger.Error(err, "Failed to get pod address")
			return nil, err
		}

		activeWorkloads, err := r.rmdClient.GetWorkloads(address)
		if err != nil {
			reqLogger.Info("Could not GET workloads.", "Error:", err)
			return nil, err
		}

		workload := rmd.FindWorkloadByName(activeWorkloads, rmdWorkloadName)
		if workload.UUID == rmdWorkloadName {
			// The reconciled workload is found to be actively running on this Node.
			if len(rmdWorkload.Spec.NodeSelector) == 0 {
				// Check if this Node still exists on the reconciled RmdWorkload Spec.
				// If not, append details to 'removedNodes'.
				nodeExistsOnRmdWorkloadSpec := false
				for _, rmdNodeName := range rmdWorkload.Spec.Nodes {
					if rmdNodeName == nodeName {
						nodeExistsOnRmdWorkloadSpec = true
					}
				}
				if !nodeExistsOnRmdWorkloadSpec {
					address, err := r.getPodAddress(nodeName)
					if err != nil {
						return nil, err
					}
					removedNodes = append(removedNodes, removedNodeInfo{nodeName, address, workload.ID})
				}
				continue
			} else {
				// Check if this Node's labels still match those specified in the
				// reconciled RmdWorkload NodeSelector field.
				// If not, append details to 'removedNodes'.
				node := &corev1.Node{}
				err := r.client.Get(context.TODO(), types.NamespacedName{Name: nodeName}, node)
				if err != nil {
					reqLogger.Error(err, "Failed to get Node")
					return nil, err
				}
				nodeLabels := labels.Set(node.GetObjectMeta().GetLabels())
				if !labels.AreLabelsInWhiteList(labels.Set(rmdWorkload.Spec.NodeSelector), nodeLabels) {
					address, err := r.getPodAddress(nodeName)
					if err != nil {
						return nil, err
					}
					removedNodes = append(removedNodes, removedNodeInfo{nodeName, address, workload.ID})
				}
			}
		}
	}
	return removedNodes, nil
}

// getPodAddress fetches the IP address and port of the desired service.
func (r *ReconcileRmdWorkload) getPodAddress(nodeName string) (string, error) {
	logger := log.WithName("getPodAddress")

	pods := &corev1.PodList{}
	err := r.client.List(context.TODO(), pods, client.MatchingLabels(client.MatchingLabels{"name": rmdPodNameConst}))
	if err != nil {
		logger.Error(err, "Failed to list RMD pods")
		return "", err
	}
	rmdPod, err := util.GetPodFromNodeName(pods, nodeName)
	if err != nil {
		rmdNode := &corev1.Node{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: nodeName}, rmdNode)
		if err != nil {
			logger.Error(err, "Failed to get node")
			return "", err
		}
		rmdPod, err = util.GetPodFromNodeAddresses(pods, rmdNode)
		if err != nil {
			return "", err
		}
	}

	var podIP string
	notFoundErr := errors.NewServiceUnavailable("pod address not available")
	if rmdPod.Status.PodIP != "" {
		podIP = rmdPod.Status.PodIP
	} else if len(rmdPod.Status.PodIPs) != 0 {
		podIP = rmdPod.Status.PodIPs[0].IP
	} else {
		return "", notFoundErr
	}
	if len(rmdPod.Spec.Containers) == 0 {
		return "", notFoundErr
	}
	if len(rmdPod.Spec.Containers[0].Ports) == 0 {
		return "", notFoundErr
	}
	addressPrefix := r.rmdClient.GetAddressPrefix()
	address := fmt.Sprintf("%s%s%s%d", addressPrefix, podIP, ":", rmdPod.Spec.Containers[0].Ports[0].ContainerPort)
	return address, nil

}

func (r *ReconcileRmdWorkload) addWorkload(address string, rmdWorkload *intelv1alpha1.RmdWorkload, nodeName string) error {
	logger := log.WithName("postWorkload")
	response, err := r.rmdClient.PostWorkload(rmdWorkload, address)
	if err != nil {
		logger.Error(err, "Failed to post workload to RMD", "Response:", response)
	}
	if len(rmdWorkload.Status.WorkloadStates) == 0 {
		workloadStates := make(map[string]intelv1alpha1.WorkloadState)
		rmdWorkload.Status.WorkloadStates = workloadStates
	}
	var workloadState = rmdWorkload.Status.WorkloadStates[nodeName]
	workloadState.Response = response
	rmdWorkload.Status.WorkloadStates[nodeName] = workloadState

	activeWorkloads, err := r.rmdClient.GetWorkloads(address)
	if err != nil {
		logger.Info("Could not GET workloads.", "Error:", err)
	}

	rmdWorkloadName := rmdWorkload.GetObjectMeta().GetName()
	workload := rmd.FindWorkloadByName(activeWorkloads, rmdWorkloadName)

	if workload.ID != "" {
		workloadState.ID = workload.ID
		workloadState.CosName = workload.CosName
		workloadState.Status = workload.Status
		rmdWorkload.Status.WorkloadStates[nodeName] = workloadState
	} else {
		// RMD could not apply the specified workload, find the corresponding pod
		// and delete it.
		err := r.deletePod(rmdWorkloadName, rmdWorkload.GetObjectMeta().GetNamespace())
		if err != nil {
			return err

		}
	}

	err = r.client.Status().Update(context.TODO(), rmdWorkload)
	if err != nil {
		logger.Error(err, "Failed to update RmdWorkload with workload ID")
		return err
	}
	return nil
}

func (r *ReconcileRmdWorkload) updateWorkload(address string, rmdWorkload *intelv1alpha1.RmdWorkload, nodeName string) error {
	logger := log.WithName("updateWorkload")
	response, err := r.rmdClient.PatchWorkload(rmdWorkload, address, rmdWorkload.Status.WorkloadStates[nodeName].ID)
	if err != nil {
		logger.Error(err, "Failed to patch workload to RMD")
		// do not requeue
	}
	if len(rmdWorkload.Status.WorkloadStates) == 0 {
		workloadStates := make(map[string]intelv1alpha1.WorkloadState)
		rmdWorkload.Status.WorkloadStates = workloadStates
	}

	var workloadStatusState = rmdWorkload.Status.WorkloadStates[nodeName]
	workloadStatusState.Response = response
	rmdWorkload.Status.WorkloadStates[nodeName] = workloadStatusState

	err = r.client.Status().Update(context.TODO(), rmdWorkload)
	if err != nil {
		logger.Error(err, "Failed to update RmdWorkload")
		return err
	}
	return nil
}

func (r *ReconcileRmdWorkload) removeWorkload(rmdWorkload *intelv1alpha1.RmdWorkload, removedNodes []removedNodeInfo) error {
	logger := log.WithName("removeWorkload")
	for _, removedNode := range removedNodes {
		err := r.rmdClient.DeleteWorkload(removedNode.rmdAddress, removedNode.workloadID)
		if err != nil {
			logger.Error(err, "Failed to delete workload from RMD")
			return err
		}
		delete(rmdWorkload.Status.WorkloadStates, removedNode.nodeName)
	}

	err := r.client.Status().Update(context.TODO(), rmdWorkload)
	if err != nil {
		logger.Error(err, "Failed to update RmdWorkload")
		return err
	}

	return nil
}

func (r *ReconcileRmdWorkload) deletePod(rmdWorkloadName string, namespace string) error {
	logger := log.WithName("deletePod")

	nameSlice := strings.Split(rmdWorkloadName, rmdWorkloadNameConst)
	podNamespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      nameSlice[0],
	}
	pod := &corev1.Pod{}

	err := r.client.Get(context.TODO(), podNamespacedName, pod)
	if err != nil {
		logger.Error(err, "Failed to get pod")
		if errors.IsNotFound(err) {
			return nil
		}
	}
	err = r.client.Delete(context.TODO(), pod)
	if err != nil {
		logger.Error(err, "Failed to delete pod")
		return err

	}
	return nil
}
