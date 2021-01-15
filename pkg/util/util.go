package util

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

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
