package rmd

import (
	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	rmdtypes "github.com/intel/rmd/modules/workload/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type rmdWorkloadTestCase struct {
	name             string
	workload         *intelv1alpha1.RmdWorkload
	expectedWorkload rmdtypes.RDTWorkLoad
}

func rdtWorkLoadTestCases() []*rmdtypes.RDTWorkLoad {
	wlds := make([]*rmdtypes.RDTWorkLoad, 0)
	wl1 := &rmdtypes.RDTWorkLoad{}

	wl1.UUID = "rmd-workload-pod-1"
	max1 := uint32(2)
	wl1.Cache.Max = &max1
	min1 := uint32(2)
	wl1.Cache.Min = &min1
	wlds = append(wlds, wl1)

	wl2 := &rmdtypes.RDTWorkLoad{}
	wl2.UUID = "rmd-workload-pod-2"
	max2 := uint32(2)
	wl2.Cache.Max = &max2
	min2 := uint32(1)
	wl2.Cache.Min = &min2
	wlds = append(wlds, wl2)

	wl3 := &rmdtypes.RDTWorkLoad{}
	wl3.UUID = "rmd-workload-pod-3"
	max3 := uint32(1)
	wl3.Cache.Max = &max3
	min3 := uint32(1)
	wl3.Cache.Min = &min3
	wlds = append(wlds, wl3)

	wl4 := &rmdtypes.RDTWorkLoad{}
	wl4.UUID = "rmd-workload-pod-4"
	max4 := uint32(2)
	wl4.Cache.Max = &max4
	min4 := uint32(2)
	wl4.Cache.Min = &min4
	ratio4 := float64(1.5)
	wl4.PState.Ratio = &ratio4
	monitoring4 := "on"
	wl4.PState.Monitoring = &monitoring4
	wlds = append(wlds, wl4)

	wl5 := &rmdtypes.RDTWorkLoad{}
	wl5.CoreIDs = []string{"0", "20"}
	wl5.UUID = "rmd-workload-pod-5"
	max5 := uint32(2)
	wl5.Policy = "gold"
	wl5.Cache.Max = &max5
	min5 := uint32(2)
	wl5.Cache.Min = &min5
	ratio5 := float64(1.5)
	wl5.PState.Ratio = &ratio5
	monitoring5 := "on"
	wl5.PState.Monitoring = &monitoring5
	wlds = append(wlds, wl5)

	return wlds
}

func rmdWorkloadTestCases() []*intelv1alpha1.RmdWorkload {
	return []*intelv1alpha1.RmdWorkload{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rmd-workload-pod-1",
			},

			Spec: intelv1alpha1.RmdWorkloadSpec{
				Cache: intelv1alpha1.Cache{
					Max: 2,
					Min: 2,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rmd-workload-pod-2",
			},

			Spec: intelv1alpha1.RmdWorkloadSpec{
				Cache: intelv1alpha1.Cache{
					Max: 2,
					Min: 1,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rmd-workload-pod-3",
			},

			Spec: intelv1alpha1.RmdWorkloadSpec{
				Cache: intelv1alpha1.Cache{
					Max: 1,
					Min: 1,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rmd-workload-pod-4",
			},

			Spec: intelv1alpha1.RmdWorkloadSpec{
				Cache: intelv1alpha1.Cache{
					Max: 2,
					Min: 2,
				},
				Pstate: intelv1alpha1.Pstate{
					Ratio:      "1.5",
					Monitoring: "on",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rmd-workload-pod-5",
			},

			Spec: intelv1alpha1.RmdWorkloadSpec{
				CoreIds: []string{"0", "20"},
				Cache: intelv1alpha1.Cache{
					Max: 2,
					Min: 2,
				},
				Pstate: intelv1alpha1.Pstate{
					Ratio:      "1.5",
					Monitoring: "on",
				},
				Policy: "gold",
			},
		},
	}
}
