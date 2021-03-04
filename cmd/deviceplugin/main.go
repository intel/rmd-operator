package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/intel/rmd-operator/pkg/rmd"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	// Device plugin settings.
	pluginMountPath      = "/var/lib/kubelet/device-plugins"
	kubeletEndpoint      = "kubelet.sock"
	pluginEndpointPrefix = "rmddp"
	resourceName         = "intel.com/l3_cache_ways"
	localHostAdd         = "127.0.0.1"
	httpPrefix           = "http"
	httpsPrefix          = "https"
	guaranteedPool       = "guaranteed"
)

type pluginManager struct {
	rmdClient   *rmd.OperatorRmdClient
	socketFile  string
	devices     map[string]*pluginapi.Device
	deviceFiles []string
	grpcServer  *grpc.Server
}

func newPluginManager() *pluginManager {
	return &pluginManager{
		rmdClient:   rmd.NewClient(),
		socketFile:  fmt.Sprintf("%s.sock", pluginEndpointPrefix),
		devices:     make(map[string]*pluginapi.Device),
		deviceFiles: []string{"/root/otherpmdevice"},
	}
}

func (pm *pluginManager) discoverResources() error {
	devices, err := pm.rmdClient.GetGuaranteedCacheWayPools()
	if err != nil {
		return err
	}
	for k, dev := range pm.devices {
		log.Printf("device[Key =%v] Value= %v\n", k, dev)
	}
	pm.devices = devices

	return nil
}

func (pm *pluginManager) GetDeviceState(DeviceName string) string {
	// TODO: Discover device health
	return pluginapi.Healthy
}

func (pm *pluginManager) Start() error {
	log.Printf("Discovering RMD guaranteed cache way[s]")
	if err := pm.discoverResources(); err != nil {
		return err
	}
	pluginEndpoint := filepath.Join(pluginapi.DevicePluginPath, pm.socketFile)
	log.Printf("Starting Device Plugin server at pluginEndpoint %v", pluginEndpoint)
	lis, err := net.Listen("unix", pluginEndpoint)
	if err != nil {
		log.Printf("Starting Device Plugin server failed with error: %v", err)
	}
	pm.grpcServer = grpc.NewServer()

	// Register all services
	pluginapi.RegisterDevicePluginServer(pm.grpcServer, pm)
	//api.RegisterCniEndpointServer(pm.grpcServer, pm)

	go pm.grpcServer.Serve(lis)

	// Wait for server to start by launching a blocking connection
	conn, err := grpc.Dial(pluginEndpoint, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		log.Printf("Could not establish connection with gRPC server. Error: %v", err)
		return err
	}
	log.Printf("Device Plugin server started serving")
	conn.Close()
	return nil
}

func (pm *pluginManager) Stop() error {
	log.Printf("Stopping Device Plugin gRPC server..")
	err := pm.cleanup()
	if err != nil {
		return err
	}
	pm.grpcServer.Stop()
	return nil
}

// Removes existing socket if exists
func (pm *pluginManager) cleanup() error {
	pluginEndpoint := filepath.Join(pluginapi.DevicePluginPath, pm.socketFile)
	if err := os.Remove(pluginEndpoint); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// Register registers as a grpc client with the kubelet.
func Register(kubeletEndpoint, pluginEndpoint, resourceName string) error {
	log.Printf("DP Registering with Kubelet..")
	conn, err := grpc.Dial(kubeletEndpoint, grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	if err != nil {
		log.Printf("Device Plugin cannot connect to Kubelet service. Error: %v", err)
		return err
	}
	defer conn.Close()
	client := pluginapi.NewRegistrationClient(conn)

	request := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     pluginEndpoint,
		ResourceName: resourceName,
	}

	if _, err = client.Register(context.Background(), request); err != nil {
		log.Printf("Device Plugin cannot register to Kubelet service. Error: %v", err)
		return err
	}
	return nil
}

// Implements DevicePlugin service functions
func (pm *pluginManager) ListAndWatch(empty *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {
	changed := true
	for {
		for id, dev := range pm.devices {
			state := pm.GetDeviceState(id)
			if dev.Health != state {
				changed = true
				dev.Health = state
				pm.devices[id] = dev
			}
		}
		if changed {
			resp := new(pluginapi.ListAndWatchResponse)
			for _, dev := range pm.devices {
				resp.Devices = append(resp.Devices, &pluginapi.Device{ID: dev.ID, Health: dev.Health, Topology: &pluginapi.TopologyInfo{Nodes: dev.Topology.Nodes}})
			}
			log.Printf("ListAndWatch, send devices response: %v", resp)
			if err := stream.Send(resp); err != nil {
				log.Printf("Cannot update device states. Error: %v", err)
				pm.Stop()
				return err
			}
		}
		changed = false
		time.Sleep(5 * time.Second)
	}
	return nil
}

func (pm *pluginManager) PreStartContainer(ctx context.Context, psRqt *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (pm *pluginManager) GetDevicePluginOptions(ctx context.Context, empty *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		PreStartRequired: false,
	}, nil
}

//Allocate passes the PCI Addr(s) as an env variable to the requesting container
func (pm *pluginManager) Allocate(ctx context.Context, rqt *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resp := new(pluginapi.AllocateResponse)
	ids := ""
	for _, container := range rqt.ContainerRequests {
		containerResp := new(pluginapi.ContainerAllocateResponse)
		for _, id := range container.DevicesIDs {
			log.Printf("DeviceID in Allocate: %v", id)
			dev, ok := pm.devices[id]
			if !ok {
				log.Printf("Invalid allocation request with non-existing device %v", id)
				return nil, fmt.Errorf("Error. Invalid allocation request with non-existing device %s", id)
			}
			if dev.Health != pluginapi.Healthy {
				log.Printf("Invalid allocation request with unhealthy device %v", id)
				return nil, fmt.Errorf("Error. Invalid allocation request with unhealthy device %s", id)
			}
			ids = ids + "{" + id + ", " + strconv.Itoa(int(dev.Topology.Nodes[0].ID)) + "}"
		}
		envmap := make(map[string]string)
		envmap[resourceName] = ids
		containerResp.Envs = envmap
		resp.ContainerResponses = append(resp.ContainerResponses, containerResp)
	}
	return resp, nil
}

func main() {
	flag.Parse()
	log.Printf("Starting Device Plugin...")
	pm := newPluginManager()
	if pm == nil {
		log.Printf("Unable to get instance")
		return
	}
	pm.cleanup()

	// respond to syscalls for termination
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Start server
	if err := pm.Start(); err != nil {
		log.Printf("Device plugin Start() failed with error %v", err)
		return
	}
	log.Printf("Started RMD device plugin...")

	// Registers with Kubelet.
	err := Register(path.Join(pluginMountPath, kubeletEndpoint), pm.socketFile, resourceName)
	if err != nil {
		log.Printf("Device Plugin failed to register with the Kubelet. Error: %v", err)
		return
	}
	log.Printf("Device Plugin registered with the Kubelet")

	// Catch termination signals
	select {
	case sig := <-sigCh:
		log.Printf("Received signal %v", sig)
		pm.Stop()
		return
	}
}
