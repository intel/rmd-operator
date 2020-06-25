package rmd

import (
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"reflect"
	"sigs.k8s.io/yaml"
	"testing"
)

func TestNewClient(t *testing.T) {
	tcases := []struct {
		name            string
		rmdPod          *corev1.Pod
		validKeyLength  bool
		expectedDefault bool
		expectedErr     bool
	}{
		{
			name: "HTTPS port",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8443,
								},
							},
						},
					},
				},
			},
			validKeyLength:  true,
			expectedDefault: false,
			expectedErr:     false,
		},
		{
			name: "HTTP port",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8081,
								},
							},
						},
					},
				},
			},
			validKeyLength:  true,
			expectedDefault: true,
			expectedErr:     false,
		},
		{
			name: "random port",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
			validKeyLength:  true,
			expectedDefault: true,
			expectedErr:     false,
		},
		{
			name: "no port",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			validKeyLength:  true,
			expectedDefault: true,
			expectedErr:     false,
		},
		{
			name: "no container",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{},
			},
			validKeyLength:  true,
			expectedDefault: true,
			expectedErr:     false,
		},
		{
			name: "HTTPS port - short key",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8443,
								},
							},
						},
					},
				},
			},
			validKeyLength:  false,
			expectedDefault: true,
			expectedErr:     true,
		},
		{
			name: "HTTP port - short key",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8081,
								},
							},
						},
					},
				},
			},
			validKeyLength:  false,
			expectedDefault: true,
			expectedErr:     true,
		},
		{
			name: "random port - short key",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
			validKeyLength:  false,
			expectedDefault: true,
			expectedErr:     true,
		},
		{
			name: "no port - short key",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			validKeyLength:  false,
			expectedDefault: true,
			expectedErr:     true,
		},
		{
			name: "no container - short key",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{},
			},
			validKeyLength:  false,
			expectedDefault: true,
			expectedErr:     true,
		},
	}
	for _, tc := range tcases {
		content, err := yaml.Marshal(tc.rmdPod)
		if err := ioutil.WriteFile("./testpod", content, 0666); err != nil {
			t.Fatalf("error writing to file (%v)", err)
		}

		if tc.validKeyLength {
			certPath = "test_certs/valid_certs/cert.pem"
			keyPath = "test_certs/valid_certs/key.pem"
			caPath = "test_certs/valid_certs/ca.pem"
		} else {
			certPath = "test_certs/invalid_certs/cert.pem"
			keyPath = "test_certs/invalid_certs/key.pem"
			caPath = "test_certs/invalid_certs/ca.pem"
		}

		expectedErr := false
		operatorRmdClient, err := NewOperatorRmdClient()
		if err != nil {
			expectedErr = true
		}

		defaultOperatorRmdClient := NewDefaultOperatorRmdClient()

		rmdPodPath = "./testpod"
		client := NewClient()

		if tc.expectedDefault {
			if !reflect.DeepEqual(client, defaultOperatorRmdClient) || tc.expectedErr != expectedErr {
				t.Errorf("Case %v - Expected %v and %v, got %v and %v", tc.name, defaultOperatorRmdClient, tc.expectedErr, client, expectedErr)
			}
		} else {
			if !reflect.DeepEqual(client, operatorRmdClient) {
				t.Errorf("Case %v - Expected %v , got %v", tc.name, operatorRmdClient, client)
			}

		}
		os.RemoveAll("./testpod")
	}

}

func TestIsTLSEnabled(t *testing.T) {
	tcases := []struct {
		name               string
		rmdPod             *corev1.Pod
		expectedTLSEnabled bool
		expectedError      bool
	}{
		{
			name: "HTTPS port",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8443,
								},
							},
						},
					},
				},
			},
			expectedTLSEnabled: true,
			expectedError:      false,
		},
		{
			name: "HTTP port",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8081,
								},
							},
						},
					},
				},
			},
			expectedTLSEnabled: false,
			expectedError:      false,
		},
		{
			name: "random port",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
			expectedTLSEnabled: false,
			expectedError:      false,
		},
		{
			name: "no port",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
			expectedTLSEnabled: false,
			expectedError:      true,
		},
		{
			name: "no container",
			rmdPod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rmd-example-node-1.com",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{},
			},
			expectedTLSEnabled: false,
			expectedError:      true,
		},
	}
	for _, tc := range tcases {
		content, err := yaml.Marshal(tc.rmdPod)
		if err := ioutil.WriteFile("./testpod", content, 0666); err != nil {
			t.Fatalf("error writing to file (%v)", err)
		}

		errorReturned := false
		tlsEnabled, err := isTLSEnabled("./testpod")
		if err != nil {
			errorReturned = true
		}
		if tc.expectedTLSEnabled != tlsEnabled || tc.expectedError != errorReturned {
			t.Errorf("Expected %v and %v, got %v and %v", tc.expectedTLSEnabled, tc.expectedError, tlsEnabled, errorReturned)
		}
		os.RemoveAll("./testpod")
	}

}
