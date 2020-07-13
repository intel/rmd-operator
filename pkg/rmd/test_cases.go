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
	wl1.Rdt.Cache.Max = &max1
	min1 := uint32(2)
	wl1.Rdt.Cache.Min = &min1
	wlds = append(wlds, wl1)

	wl2 := &rmdtypes.RDTWorkLoad{}
	wl2.UUID = "rmd-workload-pod-2"
	max2 := uint32(2)
	wl2.Rdt.Cache.Max = &max2
	min2 := uint32(1)
	wl2.Rdt.Cache.Min = &min2
	mbaPercent2 := uint32(50)
	wl2.Rdt.Mba.Percentage = &mbaPercent2
	wlds = append(wlds, wl2)

	wl3 := &rmdtypes.RDTWorkLoad{}
	wl3.UUID = "rmd-workload-pod-3"
	max3 := uint32(1)
	wl3.Rdt.Cache.Max = &max3
	min3 := uint32(1)
	wl3.Rdt.Cache.Min = &min3
	mbaMbps3 := uint32(100)
	wl3.Rdt.Mba.Mbps = &mbaMbps3
	wlds = append(wlds, wl3)

	wl4 := &rmdtypes.RDTWorkLoad{}
	wl4.UUID = "rmd-workload-pod-4"
	max4 := uint32(2)
	wl4.Rdt.Cache.Max = &max4
	min4 := uint32(2)
	wl4.Rdt.Cache.Min = &min4
	mbaPercent4 := uint32(50)
	wl4.Rdt.Mba.Percentage = &mbaPercent4
	mbaMbps4 := uint32(100)
	wl4.Rdt.Mba.Mbps = &mbaMbps4
	wl4plugins := make(map[string]map[string]interface{})
	ratio4 := float64(1.5)
	wl4pstate := make(map[string]interface{})
	wl4pstate["ratio"] = ratio4
	wl4plugins["pstate"] = wl4pstate
	monitoring4 := "on"
	wl4plugins["pstate"]["monitoring"] = monitoring4
	wl4.Plugins = wl4plugins
	wlds = append(wlds, wl4)

	wl5 := &rmdtypes.RDTWorkLoad{}
	wl5.CoreIDs = []string{"0", "20"}
	wl5.UUID = "rmd-workload-pod-5"
	max5 := uint32(2)
	wl5.Policy = "gold"
	wl5.Rdt.Cache.Max = &max5
	min5 := uint32(2)
	wl5.Rdt.Cache.Min = &min5
	mbaPercent5 := uint32(25)
	wl5.Rdt.Mba.Percentage = &mbaPercent5
	mbaMbps5 := uint32(150)
	wl5.Rdt.Mba.Mbps = &mbaMbps5
	wl5plugins := make(map[string]map[string]interface{})
	ratio5 := float64(1.5)
	wl5pstate := make(map[string]interface{})
	wl5pstate["ratio"] = ratio5
	wl5plugins["pstate"] = wl5pstate
	monitoring5 := "on"
	wl5plugins["pstate"]["monitoring"] = monitoring5
	wl5.Plugins = wl5plugins
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
				Rdt: intelv1alpha1.Rdt{
					Cache: intelv1alpha1.Cache{
						Max: 2,
						Min: 2,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rmd-workload-pod-2",
			},

			Spec: intelv1alpha1.RmdWorkloadSpec{
				Rdt: intelv1alpha1.Rdt{
					Cache: intelv1alpha1.Cache{
						Max: 2,
						Min: 1,
					},
					Mba: intelv1alpha1.Mba{
						Percentage: 50,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rmd-workload-pod-3",
			},

			Spec: intelv1alpha1.RmdWorkloadSpec{
				Rdt: intelv1alpha1.Rdt{
					Cache: intelv1alpha1.Cache{
						Max: 1,
						Min: 1,
					},
					Mba: intelv1alpha1.Mba{
						Mbps: 100,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rmd-workload-pod-4",
			},

			Spec: intelv1alpha1.RmdWorkloadSpec{
				Rdt: intelv1alpha1.Rdt{
					Cache: intelv1alpha1.Cache{
						Max: 2,
						Min: 2,
					},
					Mba: intelv1alpha1.Mba{
						Percentage: 50,
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rmd-workload-pod-5",
			},

			Spec: intelv1alpha1.RmdWorkloadSpec{
				CoreIds: []string{"0", "20"},
				Rdt: intelv1alpha1.Rdt{
					Cache: intelv1alpha1.Cache{
						Max: 2,
						Min: 2,
					},
					Mba: intelv1alpha1.Mba{
						Percentage: 25,
						Mbps:       150,
					},
				},
				Plugins: intelv1alpha1.Plugins{
					Pstate: intelv1alpha1.Pstate{
						Ratio:      "1.5",
						Monitoring: "on",
					},
				},
				Policy: "gold",
			},
		},
	}
}
