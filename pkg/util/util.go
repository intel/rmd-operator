package util

import (
	"context"
	"fmt"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"net"
	"net/url"
	"os"
	"path/filepath"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"golang.org/x/sys/unix"
)

var log = logf.Log.WithName("rmd_operator_utils")

const (
	// unixProtocol is the network protocol of unix socket.
	unixProtocol = "unix"
)

// CreateListener creates a listener on the specified endpoint.
func CreateListener(endpoint string) (net.Listener, error) {
	protocol, addr, err := parseEndpointWithFallbackProtocol(endpoint, unixProtocol)
	if err != nil {
		return nil, err
	}
	if protocol != unixProtocol {
		return nil, fmt.Errorf("only support unix socket endpoint")
	}

	// Unlink to cleanup the previous socket file.
	err = unix.Unlink(addr)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to unlink socket file %q: %v", addr, err)
	}

	if err := os.MkdirAll(filepath.Dir(addr), 0750); err != nil {
		return nil, fmt.Errorf("error creating socket directory %q: %v", filepath.Dir(addr), err)
	}

	// Create the socket on a tempfile and move it to the destination socket to handle improper cleanup
	file, err := ioutil.TempFile(filepath.Dir(addr), "")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %v", err)
	}

	if err := os.Remove(file.Name()); err != nil {
		return nil, fmt.Errorf("failed to remove temporary file: %v", err)
	}

	l, err := net.Listen(protocol, file.Name())
	if err != nil {
		return nil, err
	}

	if err = os.Rename(file.Name(), addr); err != nil {
		return nil, fmt.Errorf("failed to move temporary file to addr %q: %v", addr, err)
	}

	return l, nil
}

// GetAddressAndDialer returns the address parsed from the given endpoint and a context dialer.
func GetAddressAndDialer(endpoint string) (string, func(ctx context.Context, addr string) (net.Conn, error), error) {
	protocol, addr, err := parseEndpointWithFallbackProtocol(endpoint, unixProtocol)
	if err != nil {
		return "", nil, err
	}
	if protocol != unixProtocol {
		return "", nil, fmt.Errorf("only support unix socket endpoint")
	}

	return addr, dial, nil
}

func dial(ctx context.Context, addr string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, unixProtocol, addr)
}

func parseEndpointWithFallbackProtocol(endpoint string, fallbackProtocol string) (protocol string, addr string, err error) {
	logger := log.WithName("parseEndpointWithFallbackProtocol")

	if protocol, addr, err = parseEndpoint(endpoint); err != nil && protocol == "" {
		fallbackEndpoint := fallbackProtocol + "://" + endpoint
		protocol, addr, err = parseEndpoint(fallbackEndpoint)
		if err == nil {
			logger.Info("Using this endpoint is deprecated", "endpoint", endpoint, "please consider using full url format", fallbackEndpoint)
		}
	}
	return
}

func parseEndpoint(endpoint string) (string, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", "", err
	}

	switch u.Scheme {
	case "tcp":
		return "tcp", u.Host, nil

	case "unix":
		return "unix", u.Path, nil

	case "":
		return "", "", fmt.Errorf("using %q as endpoint is deprecated, please consider using full url format", endpoint)

	default:
		return u.Scheme, "", fmt.Errorf("protocol %q not supported", u.Scheme)
	}
}

func GetPodFromNodeName(pods *corev1.PodList, nodename string) (corev1.Pod, error) {
	for _, pod := range pods.Items {
		if nodename == pod.Spec.NodeName {
			return pod, nil
		}
	}
	return corev1.Pod{}, errors.NewServiceUnavailable(fmt.Sprintf("%s%s", "rmd pod not found by nodename ", nodename))
}

func GetPodFromNodeAddresses(pods *corev1.PodList, node *corev1.Node) (corev1.Pod, error) {
	for _, pod := range pods.Items {
		for _, address := range node.Status.Addresses {
			if address.Address == pod.Status.HostIP {
				return pod, nil
			}
		}
	}
	return corev1.Pod{}, errors.NewServiceUnavailable(fmt.Sprintf("%s%s", "rmd pod not found by addresses for node ", node.GetObjectMeta().GetName()))
}
