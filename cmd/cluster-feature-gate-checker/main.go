package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Drumato/cluster-feature-gate-checker/checker"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	ctx := context.Background()

	kubeconfigFilePath := os.Getenv("KUBECONFIG")
	if kubeconfigFilePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		kubeconfigFilePath = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigFilePath)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	componentMatrix, err := checker.CollectRunningClusterFeatureGates(ctx, clientset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}

	fmt.Println("===== k8s running cluster feature gate checker =====")

	for componentName, pods := range componentMatrix {
		fmt.Printf("### %s\n", componentName)

		for _, p := range pods {
			for _, c := range p.Containers {
				if len(c.FeatureGates) == 0 {
					fmt.Printf("%s/%s: No Feature Gates Found\n", p.Name, c.Name)
					continue
				}

				fmt.Printf("    %s/%s:\n", p.Name, c.Name)
				for k, v := range c.FeatureGates {
					fmt.Printf("        %s: %s\n", k, v)
				}
			}
		}

		fmt.Println("\n")
	}

}
