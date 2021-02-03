package nodeagent

import (
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/apis/core"
	v1qos "k8s.io/kubernetes/pkg/apis/core/v1/helper/qos"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
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
	defaultNamespace      = "default"
	rmdWorkloadNameConst  = "-rmd-workload-"
	policyConst           = "policy"
	cacheMinConst         = "cache_min"
	pstateMonitoringConst = "pstate_monitoring"
	pstateRatioConst      = "pstate_ratio"
	mbaPercentageConst    = "mba_percentage"
	mbaMbpsConst          = "mba_mbps"
	l3Cache               = "intel.com/l3_cache_ways"
	dockerPrefix          = "docker://"
	cpusetFileConst       = "/cpuset.cpus"
	kubepodsConst         = "kubepods"
)

var unifiedCgroupPath = "/sys/fs/cgroup/"
var legacyCgroupPath = "/sys/fs/cgroup/cpuset/"
var hybridCgroupPath = "/sys/fs/cgroup/unified/"

var log = logf.Log.WithName("controller_pod")

type containerInformation struct {
	coreIDs  []string
	maxCache int
}

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Pod Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePod{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("pod-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Pod
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Pod
	err = c.Watch(&source.Kind{Type: &intelv1alpha1.RmdWorkload{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &corev1.Pod{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcilePod implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcilePod{}

// ReconcilePod reconciles a Pod object
type ReconcilePod struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Pod object and makes changes based on the state read
// and what is in the Pod.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePod) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	// Fetch the Pod instance
	cachePod := &corev1.Pod{}
	err := r.client.Get(context.TODO(), request.NamespacedName, cachePod)
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
	podNotRunningErr := errors.NewServiceUnavailable("pod not in running phase")
	if cachePod.Status.Phase != corev1.PodRunning {
		reqLogger.Info("Pod not running", "pod status:", cachePod.Status.Phase)
		return reconcile.Result{}, podNotRunningErr
	}

	// Get node-agent pod to compare host node with cachePod
	nodeAgentPod, err := r.getNodeAgentPod()
	if err != nil {
		return reconcile.Result{}, err
	}
	if cachePod.Status.HostIP != nodeAgentPod.Status.HostIP {
		reqLogger.Info("Pod does not belong to same node as node agent")
		return reconcile.Result{}, nil
	}

	rmdWorkloads, err := buildRmdWorkload(cachePod)
	if err != nil {
		return reconcile.Result{}, err
	}
	if len(rmdWorkloads) == 0 {
		return reconcile.Result{}, nil
	}
	for _, rmdWorkload := range rmdWorkloads {
		rmdWorkloadName := rmdWorkload.GetObjectMeta().GetName()
		err = r.client.Get(context.TODO(), types.NamespacedName{
			Name: rmdWorkloadName, Namespace: request.Namespace}, rmdWorkload)

		if err != nil {
			if errors.IsNotFound(err) {
				// RmdWorkload not found, create it
				if err := controllerutil.SetControllerReference(cachePod, rmdWorkload, r.scheme); err != nil {
					reqLogger.Error(err, "unable to set owner reference on new rmdWorkload")
					return reconcile.Result{}, err
				}
				reqLogger.Info("Create workload for pod container requesting cache", "workload name", rmdWorkloadName)
				err = r.client.Create(context.TODO(), rmdWorkload)
				if err != nil {
					reqLogger.Error(err, "Failed to create rmdWorkload")
					return reconcile.Result{}, err
				}
				// Continue to next workload
				continue
			}
			return reconcile.Result{}, err
		}
		// RmdWorkload found, update it.
		err = r.client.Update(context.TODO(), rmdWorkload)
		if err != nil {
			reqLogger.Error(err, "Failed to update rmdWorkload")
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func (r *ReconcilePod) getNodeAgentPod() (*corev1.Pod, error) {
	logger := log.WithName("getNodeAgentPod")
	namespaceList := &corev1.NamespaceList{}
	err := r.client.List(context.TODO(), namespaceList)
	if err != nil {
		logger.Error(err, "Failed to list namespaces")
		return nil, err
	}
	nodeAgentPod := &corev1.Pod{}
	for _, namespace := range namespaceList.Items {
		namespaceName := namespace.GetObjectMeta().GetName()
		if namespaceName == core.NamespaceSystem {
			continue
		}
		if namespaceName == core.NamespacePublic {
			continue
		}
		if namespaceName == core.NamespaceNodeLease {
			continue
		}
		nodeAgentPod, err = k8sutil.GetPod(context.TODO(), r.client, namespaceName)
		if err != nil {
			if errors.IsNotFound(err) {
				logger.Info("attempt discovery in another namespace")
				continue
			}
			return nil, err
		}
		break
	}
	logger.Info("returning with node agent pod", "nodeAgentPod", nodeAgentPod)
	return nodeAgentPod, nil
}

func buildRmdWorkload(pod *corev1.Pod) ([]*intelv1alpha1.RmdWorkload, error) {
	logger := log.WithName("buildRmdWorkload")

	// Check pod for containers requesting l3 cache.
	containersRequestingCache := getContainersRequestingCache(pod)
	if len(containersRequestingCache) == 0 {
		logger.Info("No container requesting cache found in pod")
		return nil, nil
	}
	rmdWorkloads := make([]*intelv1alpha1.RmdWorkload, 0)
	for _, container := range containersRequestingCache {
		// Container name should NOT contain "-rmd-workload-" substring.
		if strings.Contains(container.Name, rmdWorkloadNameConst) {
			logger.Info("Container name must NOT contain '-rmd-workload-' substring.", "Workload will not be created for pod", pod.GetObjectMeta().GetName(), "container", container.Name)
			continue
		}

		containerInfo, err := getContainerInfo(pod, container)
		if err != nil {
			return nil, err
		}

		rmdWorkload := &intelv1alpha1.RmdWorkload{}
		//Create workload name. Convention: "<pod-name>rmd-workload-<container-name>"
		podName := string(pod.GetObjectMeta().GetName())
		rmdWorkloadName := fmt.Sprintf("%s%s%s", podName, rmdWorkloadNameConst, container.Name)
		podNamespace := pod.GetObjectMeta().GetNamespace()
		if podNamespace == "" {
			podNamespace = defaultNamespace
		}
		rmdWorkloadNamespacedName := types.NamespacedName{
			Name:      rmdWorkloadName,
			Namespace: podNamespace,
		}

		rmdWorkload.SetName(rmdWorkloadNamespacedName.Name)
		rmdWorkload.SetNamespace(rmdWorkloadNamespacedName.Namespace)
		rmdWorkload.Spec.Rdt.Cache.Max = containerInfo.maxCache
		rmdWorkload.Spec.CoreIds = containerInfo.coreIDs
		rmdWorkload.Spec.Nodes = make([]string, 0)
		rmdWorkload.Spec.Nodes = append(rmdWorkload.Spec.Nodes, pod.Spec.NodeName)
		rmdWorkload.Spec.NodeSelector = make(map[string]string, 0)
		rmdWorkload.Spec.ReservedCoreIds = make([]string, 0)

		getAnnotationInfo(rmdWorkload, pod, container.Name) //Changes workload in getAnnotationInfo()

		rmdWorkloads = append(rmdWorkloads, rmdWorkload)
	}
	return rmdWorkloads, nil
}

func getAnnotationInfo(rmdWorkload *intelv1alpha1.RmdWorkload, pod *corev1.Pod, containerName string) {
	workloadData := pod.GetObjectMeta().GetAnnotations()
	for field, data := range workloadData {
		if !strings.HasPrefix(field, containerName) {
			continue
		}
		switch {
		case strings.HasSuffix(field, policyConst):
			if data != "" {
				rmdWorkload.Spec.Policy = data
			}
		case strings.HasSuffix(field, cacheMinConst):
			minCache, err := strconv.Atoi(data)
			if err != nil {
				continue
			}
			rmdWorkload.Spec.Rdt.Cache.Min = minCache
		case strings.HasSuffix(field, mbaPercentageConst):
			mbaPercentage, err := strconv.Atoi(data)
			if err != nil {
				continue
			}
			rmdWorkload.Spec.Rdt.Mba.Percentage = mbaPercentage
		case strings.HasSuffix(field, mbaMbpsConst):
			mbaMbps, err := strconv.Atoi(data)
			if err != nil {
				continue
			}
			rmdWorkload.Spec.Rdt.Mba.Mbps = mbaMbps
		case strings.HasSuffix(field, pstateRatioConst):
			if data != "" {
				rmdWorkload.Spec.Plugins.Pstate.Ratio = data
			}
		case strings.HasSuffix(field, pstateMonitoringConst):
			if data != "" {
				rmdWorkload.Spec.Plugins.Pstate.Monitoring = data
			}
		}
	}
}

func getContainerInfo(pod *corev1.Pod, container corev1.Container) (containerInformation, error) {
	var containerInfo containerInformation //empty containerInformation struct

	logger := log.WithName("buildRmdWorkload")
	if !exclusiveCPUs(pod, &container) {
		logger.Info("Container is not requesting exclusive CPUs")
		return containerInformation{}, nil
	}
	podUID := string(pod.GetObjectMeta().GetUID())
	if podUID == "" {
		logger.Info("No pod UID found")
		return containerInformation{}, errors.NewServiceUnavailable("pod UID not found")
	}

	containerID := getContainerID(pod, container.Name)
	coreIDs, err := readCgroupCpuset(podUID, containerID)
	if err != nil {
		logger.Error(err, "failed to retrieve cpuset from cgroups")
		return containerInformation{}, err
	}
	if len(coreIDs) == 0 {
		logger.Info("cpuset not found in cgroups for container")
		return containerInformation{}, nil
	}
	containerInfo.coreIDs = coreIDs

	containerInfo.maxCache, err = getMaxCache(&container)
	if err != nil {
		return containerInformation{}, err
	}
	return containerInfo, nil
}

func getContainersRequestingCache(pod *corev1.Pod) []corev1.Container {
	containersRequestingCache := make([]corev1.Container, 0)
	for _, container := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
		for resourceName := range container.Resources.Limits {
			if resourceName.String() == l3Cache {
				containersRequestingCache = append(containersRequestingCache, container)
			}
		}
	}
	return containersRequestingCache
}

func getMaxCache(container *corev1.Container) (int, error) {
	for resourceName, limit := range container.Resources.Limits {
		if resourceName.String() == l3Cache {
			limitInt, err := strconv.Atoi(limit.String())
			if err != nil {
				return 0, err
			}
			return limitInt, nil
		}
	}
	return 0, nil
}

func getContainerID(pod *corev1.Pod, containerName string) string {
	for _, containerStatus := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
		if containerStatus.Name == containerName {
			return containerStatus.ContainerID
		}
	}
	return ""
}

func exclusiveCPUs(pod *corev1.Pod, container *corev1.Container) bool {
	if v1qos.GetPodQOS(pod) != corev1.PodQOSGuaranteed {
		return false
	}
	cpuQuantity := container.Resources.Requests[corev1.ResourceCPU]
	if cpuQuantity.Value()*1000 != cpuQuantity.MilliValue() {
		return false
	}
	return true
}

func findKubepodsCgroup() (string, error) {
	treeVersions := []string{unifiedCgroupPath, legacyCgroupPath, hybridCgroupPath}
	for _, treeVersion := range treeVersions {
		kubepodsCgroupPath, err := findCgroupPath(treeVersion, kubepodsConst)
		if err != nil {
			return "", err
		}
		if kubepodsCgroupPath != "" {
			return kubepodsCgroupPath, nil
		}
	}
	return "", nil
}

func findPodCgroup(kubepodsCgroupPath string, podUID string) (string, error) {
	podUIDUnderscores := strings.ReplaceAll(podUID, "-", "_")
	fileVersions := []string{podUID, podUIDUnderscores}
	for _, fileVersion := range fileVersions {
		podCgroupPath, err := findCgroupPath(kubepodsCgroupPath, fileVersion)
		if err != nil {
			return "", err
		}
		if podCgroupPath != "" {
			return podCgroupPath, nil
		}
	}
	return "", nil
}

func readCgroupCpuset(podUID string, containerID string) ([]string, error) {
	containerID = strings.TrimPrefix(containerID, dockerPrefix)
	kubepodsCgroupPath, err := findKubepodsCgroup()
	if err != nil {
		return nil, err
	}
	if kubepodsCgroupPath == "" {
		return nil, errors.NewServiceUnavailable("kubepods cgroup file not found")
	}

	podCgroupPath, err := findPodCgroup(kubepodsCgroupPath, podUID)
	if err != nil {
		return nil, err
	}
	if podCgroupPath == "" {
		return nil, errors.NewServiceUnavailable(fmt.Sprintf("%s%s%s%s", "podUID ", podUID, " not found in kubepods cgroup ", kubepodsCgroupPath))
	}

	containerCgroupPath, err := findCgroupPath(podCgroupPath, containerID)
	if err != nil {
		return nil, err
	}
	if containerCgroupPath == "" {
		return nil, errors.NewServiceUnavailable(fmt.Sprintf("%s%s%s%s", "containerID ", containerID, " not found in pod cgroup ", podCgroupPath))
	}

	cpusetFile := fmt.Sprintf("%s%s", containerCgroupPath, cpusetFileConst)
	cpusetBytes, err := ioutil.ReadFile(cpusetFile)
	if err != nil {
		return nil, err
	}
	cpuSetStr := strings.TrimSpace(string(cpusetBytes))
	cpuSet := cpuset.MustParse(cpuSetStr)
	cpuSetSlice := cpuSet.ToSlice()
	cpuStrSlice := make([]string, 0)
	for _, cpu := range cpuSetSlice {
		cpuStrSlice = append(cpuStrSlice, strconv.Itoa(cpu))
	}
	return cpuStrSlice, nil
}

func findCgroupPath(base string, substring string) (string, error) {
	var fullPath string
	items, err := ioutil.ReadDir(base)
	if err != nil {
		return fullPath, err
	}
	for _, item := range items {
		if strings.Contains(item.Name(), substring) {
			fullPath = fmt.Sprintf("%s%s%s", base, item.Name(), "/")
			return fullPath, nil
		}
	}
	return fullPath, nil
}
