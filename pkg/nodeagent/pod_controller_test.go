package nodeagent

import (
	"context"
	"github.com/intel/rmd-operator/pkg/apis"
	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func createReconcilePodObject(pod *corev1.Pod) (*ReconcilePod, error) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme

	// Add route Openshift scheme
	if err := apis.AddToScheme(s); err != nil {
		return nil, err
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{pod}

	// Register operator types with the runtime scheme.
	s.AddKnownTypes(intelv1alpha1.SchemeGroupVersion)

	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)

	// Create a ReconcileNode object with the scheme and fake client.
	r := &ReconcilePod{client: cl, scheme: s}

	return r, nil

}
func TestGetAnnotationInfo(t *testing.T) {
	tcases := []struct {
		name             string
		rmdWorkload      *intelv1alpha1.RmdWorkload
		pod              *corev1.Pod
		containerName    string
		expectedWorkload *intelv1alpha1.RmdWorkload
	}{
		{
			name:        "test case 1 - no errors, single container containing all annotations",
			rmdWorkload: &intelv1alpha1.RmdWorkload{},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
					UID:       "f906a249-ab9d-4180-9afa-4075e2058ac7",
					Annotations: map[string]string{
						"nginx1_policy":            "gold",
						"nginx1_mba_percentage":    "70",
						"nginx1_mba_mbps":          "100",
						"nginx1_pstate_ratio":      "1.5",
						"nginx1_pstate_monitoring": "on",
					},
				},
				Spec: corev1.PodSpec{
					NodeName: "example-node-1.com",
					Containers: []corev1.Container{
						{
							Name: "nginx1",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:        "nginx1",
							ContainerID: "7479d8c641a73fced579a3517b6d2def3f0a3a3a7e659f86ce4db61dc9f38",
						},
					},
				},
			},
			containerName: "nginx1",
			expectedWorkload: &intelv1alpha1.RmdWorkload{
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Policy: "gold",
					Rdt: intelv1alpha1.Rdt{
						Mba: intelv1alpha1.Mba{
							Percentage: 70,
							Mbps:       100,
						},
					},
					Plugins: intelv1alpha1.Plugins{
						Pstate: intelv1alpha1.Pstate{
							Ratio:      "1.5",
							Monitoring: "on",
						},
					},
				},
			},
		},
		{
			name:        "test case 2 - typos in annotations",
			rmdWorkload: &intelv1alpha1.RmdWorkload{},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
					UID:       "f906a249-ab9d-4180-9afa-4075e2058ac7",
					Annotations: map[string]string{
						"nginx1_pollicy":        "gold", //policy misspelled
						"nginx1_mba_percent":    "70",   //mba_percentage instead of mba_percentage
						"nginx1_mbs_mbps":       "100",  //mbs instead of mba
						"nginx1_pstate_ratiox":  "1.5",  //trailing 'x'
						"nginx1_pstate_monitor": "on",   //pstate_monitor instead of pstate_monitoring
					},
				},
				Spec: corev1.PodSpec{
					NodeName: "example-node-1.com",
					Containers: []corev1.Container{
						{
							Name: "nginx1",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:        "nginx1",
							ContainerID: "7479d8c641a73fced579a3517b6d2def3f0a3a3a7e659f86ce4db61dc9f38",
						},
					},
				},
			},
			containerName:    "nginx1",
			expectedWorkload: &intelv1alpha1.RmdWorkload{},
		},
		{
			name:        "test case 3 - empty string for policy annotation",
			rmdWorkload: &intelv1alpha1.RmdWorkload{},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
					UID:       "f906a249-ab9d-4180-9afa-4075e2058ac7",
					Annotations: map[string]string{
						"nginx1_policy":         "",
						"nginx1_mba_percentage": "70",
						"nginx1_mba_mbps":       "100",
					},
				},
				Spec: corev1.PodSpec{
					NodeName: "example-node-1.com",
					Containers: []corev1.Container{
						{
							Name: "nginx1",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:        "nginx1",
							ContainerID: "7479d8c641a73fced579a3517b6d2def3f0a3a3a7e659f86ce4db61dc9f38",
						},
					},
				},
			},
			containerName: "nginx1",
			expectedWorkload: &intelv1alpha1.RmdWorkload{
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Rdt: intelv1alpha1.Rdt{
						Mba: intelv1alpha1.Mba{
							Percentage: 70,
							Mbps:       100,
						},
					},
				},
			},
		},
		{
			name:        "test case 4 - incorrect type for max cache",
			rmdWorkload: &intelv1alpha1.RmdWorkload{},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
					UID:       "f906a249-ab9d-4180-9afa-4075e2058ac7",
					Annotations: map[string]string{
						"nginx1_policy":    "gold",
						"nginx1_cache_min": "2.5", //float instead of int
					},
				},
				Spec: corev1.PodSpec{
					NodeName: "example-node-1.com",
					Containers: []corev1.Container{
						{
							Name: "nginx1",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:        "nginx1",
							ContainerID: "7479d8c641a73fced579a3517b6d2def3f0a3a3a7e659f86ce4db61dc9f38",
						},
					},
				},
			},
			containerName: "nginx1",
			expectedWorkload: &intelv1alpha1.RmdWorkload{
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Policy: "gold",
				},
			},
		},
		{
			name:        "test case 5 - incorrect type for MBA Percentage",
			rmdWorkload: &intelv1alpha1.RmdWorkload{},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
					UID:       "f906a249-ab9d-4180-9afa-4075e2058ac7",
					Annotations: map[string]string{
						"nginx1_policy":         "gold",
						"nginx1_mba_percentage": "b", //char instead of int
						"nginx1_mba_mbps":       "100",
					},
				},
				Spec: corev1.PodSpec{
					NodeName: "example-node-1.com",
					Containers: []corev1.Container{
						{
							Name: "nginx1",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:        "nginx1",
							ContainerID: "7479d8c641a73fced579a3517b6d2def3f0a3a3a7e659f86ce4db61dc9f38",
						},
					},
				},
			},
			containerName: "nginx1",
			expectedWorkload: &intelv1alpha1.RmdWorkload{
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Policy: "gold",
					Rdt: intelv1alpha1.Rdt{
						Mba: intelv1alpha1.Mba{
							Mbps: 100,
						},
					},
				},
			},
		},
		{
			name:        "test case 5 - incorrect type for MBA Mbps",
			rmdWorkload: &intelv1alpha1.RmdWorkload{},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
					UID:       "f906a249-ab9d-4180-9afa-4075e2058ac7",
					Annotations: map[string]string{
						"nginx1_policy":         "gold",
						"nginx1_mba_percentage": "70",
						"nginx1_mba_mbps":       "fifty", //string instead of int
					},
				},
				Spec: corev1.PodSpec{
					NodeName: "example-node-1.com",
					Containers: []corev1.Container{
						{
							Name: "nginx1",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:        "nginx1",
							ContainerID: "7479d8c641a73fced579a3517b6d2def3f0a3a3a7e659f86ce4db61dc9f38",
						},
					},
				},
			},
			containerName: "nginx1",
			expectedWorkload: &intelv1alpha1.RmdWorkload{
				Spec: intelv1alpha1.RmdWorkloadSpec{
					Policy: "gold",
					Rdt: intelv1alpha1.Rdt{
						Mba: intelv1alpha1.Mba{
							Percentage: 70,
						},
					},
				},
			},
		},
		{
			name:        "test case 6 - annotations missing container name",
			rmdWorkload: &intelv1alpha1.RmdWorkload{},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
					UID:       "f906a249-ab9d-4180-9afa-4075e2058ac7",
					Annotations: map[string]string{
						"policy":         "gold",
						"cache_min":      "2",
						"mba_percentage": "70",
						"mba_mbps":       "100",
					},
				},
				Spec: corev1.PodSpec{
					NodeName: "example-node-1.com",
					Containers: []corev1.Container{
						{
							Name: "nginx1",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("2"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:        "nginx1",
							ContainerID: "7479d8c641a73fced579a3517b6d2def3f0a3a3a7e659f86ce4db61dc9f38",
						},
					},
				},
			},
			containerName:    "nginx1",
			expectedWorkload: &intelv1alpha1.RmdWorkload{},
		},
	}
	for _, tc := range tcases {
		getAnnotationInfo(tc.rmdWorkload, tc.pod, tc.containerName)

		if !reflect.DeepEqual(tc.rmdWorkload, tc.expectedWorkload) {
			t.Errorf("%s: Failed. Expected: %v, Got: %v\n", tc.name, tc.expectedWorkload, tc.rmdWorkload)
		}
	}
}

func TestGetContainerRequestingCache(t *testing.T) {
	tcases := []struct {
		name       string
		pod        *corev1.Pod
		containers []corev1.Container
	}{
		{
			name: "test case 1 - single container requesting cache",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "nginx1",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
								},
							},
						},
					},
				},
			},
			containers: []corev1.Container{
				{
					Name: "nginx1",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
						},
					},
				},
			},
		},

		{
			name: "test case 2 - 2 containers, 1 requesting cache",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "nginx1",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):    resource.MustParse("1"),
									corev1.ResourceName(corev1.ResourceMemory): resource.MustParse("1G"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):    resource.MustParse("1"),
									corev1.ResourceName(corev1.ResourceMemory): resource.MustParse("1G"),
								},
							},
						},

						{
							Name: "nginx2",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
								},
							},
						},
					},
				},
			},
			containers: []corev1.Container{
				{
					Name: "nginx2",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
							corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
							corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
							corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
							corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
						},
					},
				},
			},
		},
		{
			name: "test case 3 - 2 containers, 2 requesting cache.",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "nginx1",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("3"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("3"),
								},
							},
						},

						{
							Name: "nginx2",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("1"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("1"),
								},
							},
						},
					},
				},
			},
			containers: []corev1.Container{
				{
					Name: "nginx1",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
							corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
							corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("3"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
							corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
							corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("3"),
						},
					},
				},
				{
					Name: "nginx2",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
							corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
							corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("1"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
							corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
							corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("1"),
						},
					},
				},
			},
		},

		{
			name: "test case 4 - no container requesting cache",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "nginx1",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName("intel.com/qat_generic"): resource.MustParse("2"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName("intel.com/qat_generic"): resource.MustParse("2"),
								},
							},
						},
					},
				},
			},
			containers: []corev1.Container{},
		},
	}
	for _, tc := range tcases {
		containers := getContainersRequestingCache(tc.pod)
		if !reflect.DeepEqual(containers, tc.containers) {
			t.Errorf("Failed: %v - Expected \n%v\n, got \n%v\n", tc.name, tc.containers, containers)
		}

	}

}

func TestGetMaxCache(t *testing.T) {
	tcases := []struct {
		name        string
		container   *corev1.Container
		cacheLimit  int
		expectedErr bool
	}{
		{
			name: "test case 1 - container requesting 2 caches",
			container: &corev1.Container{
				Name: "nginx1",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
					},
				},
			},
			cacheLimit:  2,
			expectedErr: false,
		},

		{
			name: "test case 2 - container requests 0 cache",
			container: &corev1.Container{
				Name: "nginx2",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("0"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("0"),
					},
				},
			},
			cacheLimit:  0,
			expectedErr: false,
		},

		{
			name: "test case 3 - container requests negative amount of cache",
			container: &corev1.Container{
				Name: "nginx3",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("-2"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("-2"),
					},
				},
			},
			cacheLimit:  -2,
			expectedErr: false,
		},

		{
			name: "test case 4 - floating point number requested",
			container: &corev1.Container{
				Name: "nginx4",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2.5"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2.5"),
					},
				},
			},
			cacheLimit:  0,
			expectedErr: true,
		},

		{
			name: "test case 5 - cache requested using a resource other than l3_cache_ways",
			container: &corev1.Container{
				Name: "nginx5",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceName("intel.com/l2_cache_ways"): resource.MustParse("2"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceName("intel.com/l2_cache_ways"): resource.MustParse("2"),
					},
				},
			},
			cacheLimit:  0,
			expectedErr: false,
		},

		{
			name: "test case 6 - typos in resource used to request cache",
			container: &corev1.Container{
				Name: "nginx6",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceName("intel.com/l3_cacheways"): resource.MustParse("2"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceName("intel.com/l3_cacheways"): resource.MustParse("2"),
					},
				},
			},
			cacheLimit:  0,     //no cache can be assigned, incorrect resource
			expectedErr: false, //doesn't throw an error, just doesn't allocate cache
		},

		{
			name: "test case 7 - single typo in resource used to request minimum cache",
			container: &corev1.Container{
				Name: "nginx7",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceName("intel.com/l3_cacheways"): resource.MustParse("2"),
					},
				},
			},
			cacheLimit:  0, //no cache can be assigned 'incorrect' resource used
			expectedErr: false,
		},

		{
			name: "test case 8 - single typo in resource used to request maximum cache",
			container: &corev1.Container{
				Name: "nginx8",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceName("intel.com/l3_cacheways"): resource.MustParse("2"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("2"),
					},
				},
			},
			cacheLimit:  2, // cache will be assigned because the function looks at the cache limit not the cache request
			expectedErr: false,
		},
	}

	//loop through slice and test each testCase
	for _, tc := range tcases {
		limit, err := getMaxCache(tc.container)
		functionFailed := false
		if err != nil {
			functionFailed = true
		}
		if functionFailed != tc.expectedErr {
			t.Errorf("Failed: An error has occurred")
		}
		if limit != tc.cacheLimit {
			t.Errorf("Failed: %v \nExpected %d\n, got %d\n", tc.name, tc.cacheLimit, limit)
		}
	}
}

func TestExclusiveCPUs(t *testing.T) {
	tcases := []struct {
		name      string
		pod       *corev1.Pod
		container *corev1.Container
		expected  bool
	}{
		{
			name: "test case 1",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "nginx1",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse(".5"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("3"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse(".5"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("3"),
								},
							},
						},

						{
							Name: "nginx2",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("1"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("1"),
								},
							},
						},
					},
				},
			},
			container: &corev1.Container{
				Name: "nginx1",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse(".5"),
						corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
						corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("3"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse(".5"),
						corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
						corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("3"),
					},
				},
			},
			expected: false,
		},
		{
			name: "test case 2",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "nginx1",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("3"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("3"),
								},
							},
						},

						{
							Name: "nginx2",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("1"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
									corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
									corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("1"),
								},
							},
						},
					},
				},
			},
			container: &corev1.Container{
				Name: "nginx1",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
						corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
						corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("3"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceName(corev1.ResourceCPU):        resource.MustParse("1"),
						corev1.ResourceName(corev1.ResourceMemory):     resource.MustParse("1G"),
						corev1.ResourceName("intel.com/l3_cache_ways"): resource.MustParse("3"),
					},
				},
			},
			expected: true,
		},
	}
	for _, tc := range tcases {
		result := exclusiveCPUs(tc.pod, tc.container)
		if result != tc.expected {
			t.Errorf("Failed: %v - Expected %v, got %v", tc.name, tc.expected, result)
		}

	}

}

func TestGetNodeAgent(t *testing.T) {
	tcases := []struct {
		name                  string
		pod                   *corev1.Pod
		nodeAgentlist         *corev1.PodList
		nodeAgentpodName      string
		namespacelist         *corev1.NamespaceList
		expectedNamespaceName types.NamespacedName
	}{
		{
			name: "test case 1",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
				},
			},
			nodeAgentlist: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-node-agent-xyz",
							Namespace: "default",
						},
					},
				},
			},
			nodeAgentpodName: "rmd-node-agent-xyz",
			namespacelist: &corev1.NamespaceList{
				Items: []corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "default",
							Namespace: "default",
						},
					},
				},
			},
			expectedNamespaceName: types.NamespacedName{
				Name:      "rmd-node-agent-xyz",
				Namespace: "default",
			},
		},
		{
			name: "test case 2",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
				},
			},
			nodeAgentlist: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-node-agent-node1",
							Namespace: "default",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-node-agent-node2",
							Namespace: "default",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-node-agent-node3",
							Namespace: "default",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-node-agent-node4",
							Namespace: "default",
						},
					},
				},
			},
			nodeAgentpodName: "rmd-node-agent-node3",
			namespacelist: &corev1.NamespaceList{
				Items: []corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "default",
							Namespace: "default",
						},
					},
				},
			},
			expectedNamespaceName: types.NamespacedName{
				Name:      "rmd-node-agent-node3",
				Namespace: "default",
			},
		},
		{
			name: "test case 3",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "default",
				},
			},
			nodeAgentlist: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-node-agent-node1",
							Namespace: "default",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-node-agent-node2",
							Namespace: "test",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rmd-node-agent-node3",
							Namespace: "prod",
						},
					},
				},
			},
			nodeAgentpodName: "rmd-node-agent-node3",
			namespacelist: &corev1.NamespaceList{
				Items: []corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "default",
							Namespace: "default",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "test",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "prod",
							Namespace: "prod",
						},
					},
				},
			},
			expectedNamespaceName: types.NamespacedName{
				Name:      "rmd-node-agent-node3",
				Namespace: "prod",
			},
		},
	}
	for _, tc := range tcases {
		// Create a ReconcileNode object with the scheme and fake client.
		r, err := createReconcilePodObject(tc.pod)
		if err != nil {
			t.Fatalf("error creating ReconcilePod object: (%v)", err)
		}
		for i := range tc.namespacelist.Items {
			err = r.client.Create(context.TODO(), &tc.namespacelist.Items[i])
			if err != nil {
				t.Fatalf("could not create namespace: (%v)", err)
			}
		}
		for i := range tc.nodeAgentlist.Items {
			err = r.client.Create(context.TODO(), &tc.nodeAgentlist.Items[i])
			if err != nil {
				t.Fatalf("could not create node agent pod: (%v)", err)
			}
		}
		os.Setenv("POD_NAME", tc.nodeAgentpodName)

		nodeAgent, err := r.getNodeAgentPod()
		if err != nil {
			t.Errorf("r.getNodeAgentPod: (%v)", err)
		}

		nodeAgentNamespacedName := types.NamespacedName{
			Name:      nodeAgent.GetObjectMeta().GetName(),
			Namespace: nodeAgent.GetObjectMeta().GetNamespace(),
		}

		if !reflect.DeepEqual(nodeAgentNamespacedName, tc.expectedNamespaceName) {
			t.Errorf("Failed: %v - Expected \n%v\n, got\n %v\n", tc.name, tc.expectedNamespaceName, nodeAgentNamespacedName)

		}
	}
}
