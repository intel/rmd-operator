package rmd

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	rmdCache "github.com/intel/rmd/modules/cache"
	rmdtypes "github.com/intel/rmd/modules/workload/types"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestUpdateNodeStatusWorkload(t *testing.T) {
	wls := rdtWorkLoadTestCases()
	tcases := []struct {
		name        string
		workload    *rmdtypes.RDTWorkLoad
		expectedMap intelv1alpha1.WorkloadMap
	}{
		{
			name:     "test case 0",
			workload: wls[0],
			expectedMap: intelv1alpha1.WorkloadMap{
				"Cache Max": "2",
				"Cache Min": "2",
			},
		},
		{
			name:     "test case 1",
			workload: wls[1],
			expectedMap: intelv1alpha1.WorkloadMap{
				"Cache Max":      "2",
				"Cache Min":      "1",
				"MBA Percentage": "50",
			},
		},
		{
			name:     "test case 2",
			workload: wls[2],
			expectedMap: intelv1alpha1.WorkloadMap{
				"Cache Max": "1",
				"Cache Min": "1",
				"MBA Mbps":  "100",
			},
		},
		{
			name:     "test case 3",
			workload: wls[3],
			expectedMap: intelv1alpha1.WorkloadMap{
				"Cache Max":          "2",
				"Cache Min":          "2",
				"MBA Percentage":     "50",
				"MBA Mbps":           "100",
				"P-State Ratio":      "1.500000",
				"P-State Monitoring": "on",
			},
		},
		{
			name:     "test case 4",
			workload: wls[4],
			expectedMap: intelv1alpha1.WorkloadMap{
				"Core IDs":           "0,20",
				"Cache Max":          "2",
				"Cache Min":          "2",
				"MBA Percentage":     "25",
				"MBA Mbps":           "150",
				"P-State Ratio":      "1.500000",
				"P-State Monitoring": "on",
				"Policy":             "gold",
			},
		},
	}
	for _, tc := range tcases {
		workloadMap, err := UpdateNodeStatusWorkload(tc.workload)
		if err != nil {
			t.Errorf("error occurred: %v", err)
		}
		if !reflect.DeepEqual(workloadMap, tc.expectedMap) {
			t.Errorf("Case %v - Expected map to be %v, got %v", tc.name, tc.expectedMap, workloadMap)
		}
	}
}

func TestFormatWorkload(t *testing.T) {
	rmdWorkloads := rmdWorkloadTestCases()
	expectedRDTWorkloads := rdtWorkLoadTestCases()
	tcases := []struct {
		name             string
		workloadCR       *intelv1alpha1.RmdWorkload
		response         rmdCache.Infos
		expectedWorkload *rmdtypes.RDTWorkLoad
	}{
		{
			name:             "test case 0",
			workloadCR:       rmdWorkloads[0],
			expectedWorkload: expectedRDTWorkloads[0],
		},
		{
			name:             "test case 1",
			workloadCR:       rmdWorkloads[1],
			expectedWorkload: expectedRDTWorkloads[1],
		},
		{
			name:             "test case 2",
			workloadCR:       rmdWorkloads[2],
			expectedWorkload: expectedRDTWorkloads[2],
		},
		{
			name:             "test case 3",
			workloadCR:       rmdWorkloads[3],
			expectedWorkload: expectedRDTWorkloads[3],
		},
		{
			name:             "test case 4",
			workloadCR:       rmdWorkloads[4],
			expectedWorkload: expectedRDTWorkloads[4],
		},
		{
			name:       "test case 5",
			workloadCR: rmdWorkloads[5],
			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						ShareCPUList: "0-23,48-71",
					},
					1: {
						ShareCPUList: "24-47,72-95",
					},
				},
			},

			expectedWorkload: expectedRDTWorkloads[5],
		},
		{
			name:       "test case 6",
			workloadCR: rmdWorkloads[6],
			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						ShareCPUList: "0-23,48-71",
					},
					1: {
						ShareCPUList: "24-47,72-95",
					},
				},
			},

			expectedWorkload: expectedRDTWorkloads[6],
		},
		{
			name:       "test case 7",
			workloadCR: rmdWorkloads[7],
			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						ShareCPUList: "0-23,48-71",
					},
					1: {
						ShareCPUList: "24-47,72-95",
					},
				},
			},

			expectedWorkload: expectedRDTWorkloads[7],
		},
		{
			name:       "test case ",
			workloadCR: rmdWorkloads[8],
			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						ShareCPUList: "0-23,48-71",
					},
					1: {
						ShareCPUList: "24-47,72-95",
					},
				},
			},

			expectedWorkload: expectedRDTWorkloads[8],
		},
	}

	for _, tc := range tcases {
		// create a listener with the desired port.
		address := "127.0.0.1:8080"
		l, err := net.Listen("tcp", address)
		if err != nil {
			t.Fatalf("Failed to create listener")
		}

		ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, err := json.Marshal(tc.response)
			if err == nil {
				fmt.Fprintln(w, string(b[:]))
			}
		}))

		ts.Listener.Close()
		ts.Listener = l

		// Start the server.
		ts.Start()

		client := NewDefaultOperatorRmdClient()

		workload, err := client.formatWorkload(tc.workloadCR, ts.URL)
		if err != nil {
			t.Errorf("Error: %v", err)
		}
		if !reflect.DeepEqual(workload, tc.expectedWorkload) {
			t.Errorf("Test case: %v, expected wl to be %v, got %v", tc.name, tc.expectedWorkload, workload)
		}
		ts.Close()
	}
}

func TestGetAvailableCacheWays(t *testing.T) {
	tcases := []struct {
		name              string
		address           string
		response          rmdCache.Infos
		expectedCacheWays int64
	}{
		{
			name: "test case 1",
			response: rmdCache.Infos{
				Num:    0,
				Caches: nil,
			},
			expectedCacheWays: 0,
		},
		{
			name: "test case 2",
			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						AvailableWays: "7ff",
					},
				},
			},
			expectedCacheWays: 2047,
		},
		{
			name: "test case 3",
			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						AvailableWays: "7ff",
					},
					1: {
						AvailableWays: "7ff",
					},
				},
			},
			expectedCacheWays: 4094,
		},
		{
			name: "test case 4",
			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						AvailableWays: "7e0",
					},
					1: {
						AvailableWays: "7e0",
					},
				},
			},
			expectedCacheWays: 4032,
		},
		{
			name: "test case 5",
			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						AvailableWays: "0",
					},
					1: {
						AvailableWays: "7e0",
					},
				},
			},
			expectedCacheWays: 2016,
		},
	}

	for _, tc := range tcases {
		// create a listener with the desired port.
		address := "127.0.0.1:8080"
		l, err := net.Listen("tcp", address)
		if err != nil {
			t.Fatalf("Failed to create listener")
		}

		ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, err := json.Marshal(tc.response)
			if err == nil {
				fmt.Fprintln(w, string(b[:]))
			}
		}))

		ts.Listener.Close()
		ts.Listener = l

		// Start the server.
		ts.Start()

		client := NewDefaultOperatorRmdClient()
		ways, err := client.GetAvailableCacheWays(ts.URL)
		if err != nil {
			t.Fatalf("Error occurred when calling GetAvailableCacheWays")
		}
		if ways != tc.expectedCacheWays {
			t.Errorf("Failed %v, expected: %v, got %v", tc.name, tc.expectedCacheWays, ways)
		}

		ts.Close()
	}
}

func TestGetAllCPUs(t *testing.T) {
	tcases := []struct {
		name            string
		address         string
		response        rmdCache.Infos
		expectedAllCPUs string
	}{
		{
			name: "test case 1",
			response: rmdCache.Infos{
				Num:    0,
				Caches: nil,
			},
			expectedAllCPUs: "",
		},
		{
			name: "test case 2",
			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						ShareCPUList: "0-10",
					},
				},
			},
			expectedAllCPUs: "0-10",
		},
		{
			name: "test case 3",
			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						ShareCPUList: "0-10",
					},
					1: {
						ShareCPUList: "11-20",
					},
				},
			},
			expectedAllCPUs: "0-20",
		},
		{
			name: "test case 4",
			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						ShareCPUList: "0-23,48-71",
					},
					1: {
						ShareCPUList: "24-47,72-95",
					},
				},
			},
			expectedAllCPUs: "0-95",
		},
		{
			name: "test case 5",
			response: rmdCache.Infos{
				Num: 0,
				Caches: map[uint32]rmdCache.Info{
					0: {
						ShareCPUList: "0-9,21-29",
					},
					1: {
						ShareCPUList: "10-19,30-39",
					},
				},
			},
			expectedAllCPUs: "0-19,21-39",
		},
	}

	for _, tc := range tcases {
		// create a listener with the desired port.
		address := "127.0.0.1:8080"
		l, err := net.Listen("tcp", address)
		if err != nil {
			t.Fatalf("Failed to create listener")
		}

		ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, err := json.Marshal(tc.response)
			if err == nil {
				fmt.Fprintln(w, string(b[:]))
			}
		}))

		ts.Listener.Close()
		ts.Listener = l

		// Start the server.
		ts.Start()

		client := NewDefaultOperatorRmdClient()
		cpus, err := client.getAllCPUs(ts.URL)
		if err != nil {
			t.Fatalf("Error occurred when calling GetAvailableCacheWays")
		}
		if cpus != tc.expectedAllCPUs {
			t.Errorf("Failed %v, expected: %v, got %v", tc.name, tc.expectedAllCPUs, cpus)
		}

		ts.Close()

	}
}

func TestFindWorkloadByName(t *testing.T) {
	wls := rdtWorkLoadTestCases()
	tcases := []struct {
		name             string
		workloads        []*rmdtypes.RDTWorkLoad
		workloadName     string
		expectedWorkload *rmdtypes.RDTWorkLoad
	}{
		{
			name:             "test case 1",
			workloads:        wls,
			workloadName:     "rmd-workload-pod-1",
			expectedWorkload: wls[0],
		},
		{
			name:             "test case 2",
			workloads:        wls,
			workloadName:     "rmd-workload-pod-2",
			expectedWorkload: wls[1],
		},
		{
			name:             "test case 3",
			workloads:        wls,
			workloadName:     "rmd-workload-pod-3",
			expectedWorkload: wls[2],
		},
		{
			name:             "test case 4",
			workloads:        wls,
			workloadName:     "rmd-workload-pod-4",
			expectedWorkload: wls[3],
		},
		{
			name:             "test case 5",
			workloads:        wls,
			workloadName:     "rmd-workload-pod-5",
			expectedWorkload: wls[4],
		},
		{
			name:             "test case 6",
			workloads:        wls,
			workloadName:     "rmd-workload-pod-x",
			expectedWorkload: &rmdtypes.RDTWorkLoad{},
		},
		{
			name:             "test case 7",
			workloads:        nil,
			workloadName:     "rmd-workload-pod-x",
			expectedWorkload: &rmdtypes.RDTWorkLoad{},
		},
	}
	for _, tc := range tcases {
		workload := FindWorkloadByName(tc.workloads, tc.workloadName)
		if !reflect.DeepEqual(workload, tc.expectedWorkload) {
			t.Errorf("Expected %v, got %v", tc.expectedWorkload, workload)
		}
	}
}

func TestVerifyKeyLength(t *testing.T) {
	tcases := []struct {
		name        string
		certPath    string
		keyPath     string
		expectedErr bool
	}{
		{
			name:        "valid cert and key",
			certPath:    "test_certs/valid_certs/cert.pem",
			keyPath:     "test_certs/valid_certs/key.pem",
			expectedErr: false,
		},
		{
			name:        "invalid cert and key",
			certPath:    "test_certs/invalid_certs/cert.pem",
			keyPath:     "test_certs/invalid_certs/key.pem",
			expectedErr: true,
		},
	}
	for _, tc := range tcases {
		cert, err := tls.LoadX509KeyPair(tc.certPath, tc.keyPath)
		if err != nil {
			t.Fatalf("error loading key pair")
		}
		expectedErr := false
		err = verifyKeyLength(cert)
		if err != nil {
			expectedErr = true
		}

		if expectedErr != tc.expectedErr {
			t.Errorf("Case %v:  Expected %v, got %v", tc.name, tc.expectedErr, expectedErr)
		}
	}
}
