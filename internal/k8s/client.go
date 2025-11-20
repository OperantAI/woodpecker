/*
Copyright 2023 Operant AI
*/
package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/operantai/woodpecker/internal/output"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Client struct {
	Clientset  *kubernetes.Clientset
	RestConfig *rest.Config
}

func NewClient() (*Client, error) {
	kubeconfig := getKubeConfigPath()

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Kubernetes Client: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Kubernetes Client: %w", err)
	}

	return &Client{
		Clientset:  clientset,
		RestConfig: config,
	}, nil
}

// getKubeConfigPath returns the path to the kubeconfig file by checking the KUBECONFIG
// environment variable or defaulting to ~/.kube/config
func getKubeConfigPath() string {
	// Create a Kubernetes Client
	var kubeconfig string
	kubeconfig, _ = os.LookupEnv("KUBECONFIG")
	if kubeconfig != "" {
		return kubeconfig
	}
	if home := homedir.HomeDir(); home != "" {
		// check if ~/.kube/config file exists
		_, error := os.Stat(filepath.Join(home, ".kube", "config"))
		if os.IsNotExist(error) {
			output.WriteError("~/.kube/config file does not exist")
		} else {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}
	return kubeconfig
}

func NewClientInContainer() (*Client, error) {

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		Clientset: clientset,
	}, nil
}
