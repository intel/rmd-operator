package rmd

import (
	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	rmdtypes "github.com/intel/rmd/modules/workload/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	wl6 := &rmdtypes.RDTWorkLoad{}
	wl6.CoreIDs = []string{"0-95"}
	wl6.UUID = "rmd-workload-pod-6"
	max6 := uint32(2)
	wl6.Policy = "gold"
	wl6.Rdt.Cache.Max = &max6
	min6 := uint32(2)
	wl6.Rdt.Cache.Min = &min6
	mbaPercent6 := uint32(25)
	wl6.Rdt.Mba.Percentage = &mbaPercent6
	mbaMbps6 := uint32(150)
	wl6.Rdt.Mba.Mbps = &mbaMbps6
	wl6plugins := make(map[string]map[string]interface{})
	ratio6 := float64(1.5)
	wl6pstate := make(map[string]interface{})
	wl6pstate["ratio"] = ratio6
	wl6plugins["pstate"] = wl5pstate
	monitoring6 := "on"
	wl6plugins["pstate"]["monitoring"] = monitoring6
	wl6.Plugins = wl6plugins
	wlds = append(wlds, wl6)

	wl7 := &rmdtypes.RDTWorkLoad{}
	wl7.CoreIDs = []string{"0-95"}
	wl7.UUID = "rmd-workload-pod-7"
	max7 := uint32(2)
	wl7.Policy = "gold"
	wl7.Rdt.Cache.Max = &max7
	min7 := uint32(2)
	wl7.Rdt.Cache.Min = &min7
	mbaPercent7 := uint32(25)
	wl7.Rdt.Mba.Percentage = &mbaPercent7
	mbaMbps7 := uint32(150)
	wl7.Rdt.Mba.Mbps = &mbaMbps7
	wl7plugins := make(map[string]map[string]interface{})
	ratio7 := float64(1.5)
	wl7pstate := make(map[string]interface{})
	wl7pstate["ratio"] = ratio7
	wl7plugins["pstate"] = wl7pstate
	monitoring7 := "on"
	wl7plugins["pstate"]["monitoring"] = monitoring7
	wl7.Plugins = wl7plugins
	wlds = append(wlds, wl7)

	wl8 := &rmdtypes.RDTWorkLoad{}
	wl8.CoreIDs = []string{"10-95"}
	wl8.UUID = "rmd-workload-pod-8"
	max8 := uint32(2)
	wl8.Policy = "gold"
	wl8.Rdt.Cache.Max = &max8
	min8 := uint32(2)
	wl8.Rdt.Cache.Min = &min8
	mbaPercent8 := uint32(25)
	wl8.Rdt.Mba.Percentage = &mbaPercent8
	mbaMbps8 := uint32(150)
	wl8.Rdt.Mba.Mbps = &mbaMbps8
	wl8plugins := make(map[string]map[string]interface{})
	ratio8 := float64(1.5)
	wl8pstate := make(map[string]interface{})
	wl8pstate["ratio"] = ratio8
	wl8plugins["pstate"] = wl8pstate
	monitoring8 := "on"
	wl8plugins["pstate"]["monitoring"] = monitoring8
	wl8.Plugins = wl8plugins
	wlds = append(wlds, wl8)

	wl9 := &rmdtypes.RDTWorkLoad{}
	wl9.CoreIDs = []string{"0,3,8-95"}
	wl9.UUID = "rmd-workload-pod-9"
	max9 := uint32(2)
	wl9.Policy = "gold"
	wl9.Rdt.Cache.Max = &max9
	min9 := uint32(2)
	wl9.Rdt.Cache.Min = &min9
	mbaPercent9 := uint32(25)
	wl9.Rdt.Mba.Percentage = &mbaPercent9
	mbaMbps9 := uint32(150)
	wl9.Rdt.Mba.Mbps = &mbaMbps9
	wl9plugins := make(map[string]map[string]interface{})
	ratio9 := float64(1.5)
	wl9pstate := make(map[string]interface{})
	wl9pstate["ratio"] = ratio9
	wl9plugins["pstate"] = wl9pstate
	monitoring9 := "on"
	wl9plugins["pstate"]["monitoring"] = monitoring9
	wl9.Plugins = wl9plugins
	wlds = append(wlds, wl9)

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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rmd-workload-pod-6",
			},

			Spec: intelv1alpha1.RmdWorkloadSpec{
				AllCores: true,
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rmd-workload-pod-7",
			},

			Spec: intelv1alpha1.RmdWorkloadSpec{
				AllCores: true,
				CoreIds:  []string{"1-2", "5", "11"},
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rmd-workload-pod-8",
			},

			Spec: intelv1alpha1.RmdWorkloadSpec{
				AllCores:        true,
				ReservedCoreIds: []string{"0-9"},
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rmd-workload-pod-9",
			},

			Spec: intelv1alpha1.RmdWorkloadSpec{
				AllCores:        true,
				ReservedCoreIds: []string{"1", "2", "4-7"},
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
