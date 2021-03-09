package rmd

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	intelv1alpha1 "github.com/intel/rmd-operator/pkg/apis/intel/v1alpha1"
	rmdCache "github.com/intel/rmd/modules/cache"
	rmdtypes "github.com/intel/rmd/modules/workload/types"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"net/http"
	"reflect"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"strings"
)

var log = logf.Log.WithName("rmd")

const (
	postResponse    = 201
	patchedResponse = 200
	post            = "POST"
	patch           = "PATCH"
	deleteConst     = "DELETE"
	httpPrefix      = "http://"
	httpsPrefix     = "https://"
	tlsServerName   = "rmd-nameserver"
	localHostAdd    = "127.0.0.1"
	guaranteedPool  = "guaranteed"
)

var certPath = "/etc/certs/public/cert.pem"
var keyPath = "/etc/certs/private/key.pem"
var caPath = "/etc/certs/public/ca.pem"

// OperatorRmdClient is used by the operator to become a client to RMD
type OperatorRmdClient struct {
	client *http.Client
}

// NewOperatorRmdClient returns a TLS client to RMD
func NewOperatorRmdClient() (*OperatorRmdClient, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return &OperatorRmdClient{}, err
	}
	err = verifyKeyLength(cert)
	if err != nil {
		return &OperatorRmdClient{}, err
	}
	caCert, err := ioutil.ReadFile(caPath)
	if err != nil {
		return &OperatorRmdClient{}, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
		ServerName:   tlsServerName,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	rmdClient := &OperatorRmdClient{
		client: client,
	}
	return rmdClient, nil
}

func verifyKeyLength(cert tls.Certificate) error {
	var keyLength int
	switch privKey := cert.PrivateKey.(type) {
	case *rsa.PrivateKey:
		keyLength = privKey.N.BitLen()
	case *ecdsa.PrivateKey:
		keyLength = privKey.Curve.Params().BitSize
	default:
		return errors.NewBadRequest("unsupported private key")
	}
	if keyLength < 2048 {
		return errors.NewBadRequest("key length less than 2048")
	}
	return nil
}

// NewDefaultOperatorRmdClient returns a default client for testing and debugging
func NewDefaultOperatorRmdClient() *OperatorRmdClient {
	defaultClient := &http.Client{}
	rmdClient := &OperatorRmdClient{
		client: defaultClient,
	}
	return rmdClient
}

// UpdateNodeStatusWorkload populates WorkloadMap with workload data for RmdNodeState
func UpdateNodeStatusWorkload(workload *rmdtypes.RDTWorkLoad) (intelv1alpha1.WorkloadMap, error) {
	workloadMap := make(intelv1alpha1.WorkloadMap)

	if workload.ID != "" {
		workloadMap["ID"] = workload.ID
	}
	if len(workload.CoreIDs) != 0 {
		workloadMap["Core IDs"] = strings.Join(workload.CoreIDs, ",")
	}
	if workload.Status != "" {
		workloadMap["Status"] = workload.Status
	}
	if workload.CosName != "" {
		workloadMap["Cos Name"] = workload.CosName
	}
	if workload.Rdt.Cache.Max != nil {
		workloadMap["Cache Max"] = strconv.Itoa(int(*workload.Rdt.Cache.Max))
	}
	if workload.Rdt.Cache.Min != nil {
		workloadMap["Cache Min"] = strconv.Itoa(int(*workload.Rdt.Cache.Min))
	}
	if workload.Rdt.Mba.Percentage != nil {
		workloadMap["MBA Percentage"] = strconv.Itoa(int(*workload.Rdt.Mba.Percentage))
	}
	if workload.Rdt.Mba.Mbps != nil {
		workloadMap["MBA Mbps"] = strconv.Itoa(int(*workload.Rdt.Mba.Mbps))
	}
	if workload.Origin != "" {
		workloadMap["Origin"] = workload.Origin
	}
	if workload.Policy != "" {
		workloadMap["Policy"] = workload.Policy
	}
	if len(workload.Plugins) != 0 {
		pluginsData, err := json.Marshal(workload.Plugins)
		if err != nil {
			return nil, err
		}
		pluginsMap := make(map[string]map[string]interface{})
		err = json.Unmarshal(pluginsData, &pluginsMap)
		if err != nil {
			return nil, err
		}
		// Look for pstate data and add to workloadMap
		if pstateMap, ok := pluginsMap["pstate"]; ok {
			if ratio, ok := pstateMap["ratio"]; ok {
				if ratio != nil {
					workloadMap["P-State Ratio"] = fmt.Sprintf("%f", ratio)
				}
			}
			if monitoring, ok := pstateMap["monitoring"]; ok {
				if monitoring != nil {
					workloadMap["P-State Monitoring"] = fmt.Sprintf("%v", monitoring)
				}
			}
		}
	}
	return workloadMap, nil
}

// GetGuaranteedCacheWayPools returns available l3 cache ways for Node Status update
func (rc *OperatorRmdClient) GetGuaranteedCacheWayPools() (map[string]*pluginapi.Device, error) {
	devices := make(map[string]*pluginapi.Device)
	addressPrefix := rc.GetAddressPrefix()
	var address string
	if addressPrefix == httpPrefix {
		address = fmt.Sprintf("%s%s%s%d", addressPrefix, localHostAdd, ":", 8081)
	} else if addressPrefix == httpsPrefix {
		address = fmt.Sprintf("%s%s%s%d", addressPrefix, localHostAdd, ":", 8443)
	}

	httpString := fmt.Sprintf("%s%s", address, "/v1/cache/l3")
	resp, err := rc.client.Get(httpString)
	if err != nil {
		return devices, err
	}

	receivedJSON, err := ioutil.ReadAll(resp.Body) //This reads raw request body
	if err != nil {
		return devices, err
	}
	allCacheInfo := rmdCache.Infos{}
	err = json.Unmarshal([]byte(receivedJSON), &allCacheInfo)
	if err != nil {
		return devices, err
	}

	for _, cache := range allCacheInfo.Caches {
		var cacheWaysSlice []int
		for pool, cacheWays := range cache.AvailableWaysPool {
			if pool == guaranteedPool {
				cacheWaysSlice = cpuset.MustParse(cacheWays).ToSlice()
			}
		}
		for _, cacheWay := range cacheWaysSlice {
			numaNode, err := strconv.Atoi(cache.Node)
			if err != nil {
				return devices, err
			}
			dev := pluginapi.Device{ID: fmt.Sprintf("%s%d", cache.Node, cacheWay), Health: pluginapi.Healthy, Topology: &pluginapi.TopologyInfo{Nodes: []*pluginapi.NUMANode{{ID: int64(numaNode)}}}}
			devices[fmt.Sprintf("%s%d", cache.Node, cacheWay)] = &dev
		}
	}
	return devices, nil

}

// GetAvailableCacheWays returns available l3 cache ways for Node Status update
func (rc *OperatorRmdClient) GetAvailableCacheWays(address string) (int64, error) {
	logger := log.WithName("GetAvailableCacheWays")

	httpString := fmt.Sprintf("%s%s", address, "/v1/cache/l3")
	resp, err := rc.client.Get(httpString)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	receivedJSON, err := ioutil.ReadAll(resp.Body) //This reads raw request body
	if err != nil {
		return 0, err
	}
	allCacheInfo := rmdCache.Infos{}
	err = json.Unmarshal([]byte(receivedJSON), &allCacheInfo)
	if err != nil {
		return 0, err
	}

	var availableWays int64
	for _, cache := range allCacheInfo.Caches {
		availableWaysTemp, err := strconv.ParseInt(cache.AvailableWays, 16, 64)
		if err != nil {
			return 0, err
		}
		availableWays = availableWays + availableWaysTemp
	}
	logger.Info("Total available cache ways discovered", "available_ways", availableWays)
	return availableWays, nil
}

// GetAllCPUS returns available l3 cache ways for Node Status update
func (rc *OperatorRmdClient) getAllCPUs(address string) (string, error) {
	logger := log.WithName("getAllCPUs")

	httpString := fmt.Sprintf("%s%s", address, "/v1/cache/l3")
	resp, err := rc.client.Get(httpString)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	receivedJSON, err := ioutil.ReadAll(resp.Body) //This reads raw request body
	if err != nil {
		return "", err
	}
	allCacheInfo := rmdCache.Infos{}
	err = json.Unmarshal([]byte(receivedJSON), &allCacheInfo)
	if err != nil {
		return "", err
	}
	hostCPUSet := cpuset.NewCPUSet()
	for _, cache := range allCacheInfo.Caches {
		cacheCPUSet, err := cpuset.Parse(cache.ShareCPUList)
		if err != nil {
			return "", err
		}
		hostCPUSet = hostCPUSet.Union(cacheCPUSet)
	}
	logger.Info("All CPUs discovered on host", "hostCPUSet", hostCPUSet.String())
	return hostCPUSet.String(), nil
}

// GetWorkloads returns all active workloads on RMD instance
func (rc *OperatorRmdClient) GetWorkloads(address string) ([]*rmdtypes.RDTWorkLoad, error) {
	httpString := fmt.Sprintf("%s%s", address, "/v1/workloads")
	resp, err := rc.client.Get(httpString)
	if err != nil {
		return nil, err
	}
	receivedJSON, err := ioutil.ReadAll(resp.Body) //This reads raw request body
	if err != nil {
		return nil, err
	}
	allWorkloads := make([]*rmdtypes.RDTWorkLoad, 0)
	err = json.Unmarshal([]byte(receivedJSON), &allWorkloads)
	if err != nil {
		return nil, err
	}

	resp.Body.Close()
	return allWorkloads, nil
}

// Format Workload to rmdtypes.RDTWorkLoad{} as the workloadCR contains unnecessary fields which can
// be problematic if marshalled directly and delivered to RMD.
func (rc *OperatorRmdClient) formatWorkload(workloadCR *intelv1alpha1.RmdWorkload, address string) (*rmdtypes.RDTWorkLoad, error) {
	rdtWorkload := &rmdtypes.RDTWorkLoad{}
	rdtWorkload.UUID = workloadCR.GetObjectMeta().GetName()
	rdtWorkload.Policy = workloadCR.Spec.Policy

	// If AllCores has been declared in the workload spec, discover all cores on the host
	// and add to request.
	if workloadCR.Spec.AllCores {
		allCores, err := rc.getAllCPUs(address)
		if err != nil {
			return &rmdtypes.RDTWorkLoad{}, err
		}

		if len(workloadCR.Spec.ReservedCoreIds) != 0 {
			rsvdCPUSet := cpuset.MustParse(strings.Join(workloadCR.Spec.ReservedCoreIds, ","))
			allCoresCPUSet := cpuset.MustParse(allCores)
			finalCPUSet := allCoresCPUSet.Difference(rsvdCPUSet)
			allCores = finalCPUSet.String()
		}
		coreIDs := []string{allCores}
		rdtWorkload.CoreIDs = coreIDs
	} else {
		rdtWorkload.CoreIDs = workloadCR.Spec.CoreIds
	}

	// Add Cache data if it has been specified in the workload.
	maxCache := uint32(workloadCR.Spec.Rdt.Cache.Max)
	rdtWorkload.Rdt.Cache.Max = &maxCache
	minCache := uint32(workloadCR.Spec.Rdt.Cache.Min)
	rdtWorkload.Rdt.Cache.Min = &minCache

	// Add MBA data if it has been specified in the workload.
	mbaPercentage := uint32(workloadCR.Spec.Rdt.Mba.Percentage)
	if mbaPercentage != 0 {
		rdtWorkload.Rdt.Mba.Percentage = &mbaPercentage
	}
	mbaMbps := uint32(workloadCR.Spec.Rdt.Mba.Mbps)
	if mbaMbps != 0 {
		rdtWorkload.Rdt.Mba.Mbps = &mbaMbps
	}

	// Add Plugins (PState) data to be marshalled if it has been specified in the workload.
	plugins := make(map[string]map[string]interface{})
	pstate := make(map[string]interface{})
	if len(workloadCR.Spec.Plugins.Pstate.Ratio) != 0 {
		ratio, err := strconv.ParseFloat(workloadCR.Spec.Plugins.Pstate.Ratio, 64)
		if err != nil {
			return &rmdtypes.RDTWorkLoad{}, err
		}
		if ratio != 0 {
			pstate["ratio"] = ratio
			plugins["pstate"] = pstate
		}
	}
	if len(workloadCR.Spec.Plugins.Pstate.Monitoring) != 0 {
		monitoring := workloadCR.Spec.Plugins.Pstate.Monitoring
		if len(monitoring) != 0 {
			pstate["monitoring"] = monitoring
			plugins["pstate"] = pstate
		}
	}
	if len(plugins) != 0 {
		rdtWorkload.Plugins = plugins
	}
	return rdtWorkload, nil
}

// PostWorkload posts workload data from RmdWorkload to RMD
func (rc *OperatorRmdClient) PostWorkload(workloadCR *intelv1alpha1.RmdWorkload, address string) (string, error) {
	postFailedErr := errors.NewServiceUnavailable("Response status code error")

	data, err := rc.formatWorkload(workloadCR, address)
	if err != nil {
		return "", err
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return "Failed to marshal payload data", err
	}
	body := bytes.NewReader(payloadBytes)

	httpString := fmt.Sprintf("%s%s", address, "/v1/workloads")
	req, err := http.NewRequest(post, httpString, body)
	if err != nil {
		return "Failed to create new http post request", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := rc.client.Do(req)
	if err != nil {
		return "Failed to set header for http post request", err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return "Failed to read response to buffer", err
	}
	respStr := buf.String()

	if resp.StatusCode != postResponse && resp.StatusCode != patchedResponse {
		errStr := fmt.Sprintf("%s%v", "Fail: ", respStr)
		return errStr, postFailedErr
	}
	defer resp.Body.Close()

	successStr := fmt.Sprintf("%s%v", "Success: ", resp.StatusCode)

	return successStr, nil
}

// PatchWorkload patches workload running on RMD with workload data from RmdWorkload
func (rc *OperatorRmdClient) PatchWorkload(workloadCR *intelv1alpha1.RmdWorkload, address string, workloadID string) (string, error) {
	patchFailedErr := errors.NewServiceUnavailable("Response status code error")
	data, err := rc.formatWorkload(workloadCR, address)
	if err != nil {
		return "", err
	}

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return "Failed to marshal payload data", err
	}
	body := bytes.NewReader(payloadBytes)

	httpString := fmt.Sprintf("%s%s%s", address, "/v1/workloads/", workloadID)
	req, err := http.NewRequest(patch, httpString, body)
	if err != nil {
		return "Failed to create new http patch request", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := rc.client.Do(req)
	if err != nil {
		return "Failed to set header for http patch request", err
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return "Failed to read response to buffer", err
	}
	respStr := buf.String()
	if resp.StatusCode != patchedResponse {
		errStr := fmt.Sprintf("%s%v", "Fail: ", respStr)
		return errStr, patchFailedErr
	}
	defer resp.Body.Close()

	successStr := fmt.Sprintf("%s%v", "Success: ", resp.StatusCode)
	return successStr, nil
}

// DeleteWorkload deletes workload from RMD by workload ID
func (rc *OperatorRmdClient) DeleteWorkload(address string, workloadID string) error {
	deleteFailedErr := errors.NewServiceUnavailable("Response status code error")
	httpString := fmt.Sprintf("%s%s%s", address, "/v1/workloads/", workloadID)
	req, err := http.NewRequest(deleteConst, httpString, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := rc.client.Do(req)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != patchedResponse {
		return deleteFailedErr
	}

	defer resp.Body.Close()
	return nil
}

// FindWorkloadByName discovers a particular workload running on RMD by name/UUID
func FindWorkloadByName(workloads []*rmdtypes.RDTWorkLoad, workloadName string) *rmdtypes.RDTWorkLoad {
	for _, workload := range workloads {
		if workload.UUID == workloadName {
			return workload
		}
	}
	return &rmdtypes.RDTWorkLoad{}
}

// GetAddressPrefix returns correct address prefix based on rmdClient
func (rc *OperatorRmdClient) GetAddressPrefix() string {
	if reflect.DeepEqual(rc.client, http.DefaultClient) {
		return httpPrefix
	}
	return httpsPrefix
}
