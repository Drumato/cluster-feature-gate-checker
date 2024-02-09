package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	mapKeyKubeAPIServer = "kube-apiserver"
	mapKeyKubeScheduler = "kube-scheduler"
)

func main() {
	ctx := context.Background()
	kubeconfigFilePath := os.Getenv("KUBECONFIG")
	if kubeconfigFilePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err.Error())
		}
		kubeconfigFilePath = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigFilePath)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	pods, err := clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	podMap := constructSystemComponentPodsMap(pods)

	fmt.Println("===== k8s running cluster feature gate checker =====")

	for componentName, pods := range podMap {
		fmt.Printf("### %s\n", componentName)

		for _, p := range pods {
			for _, c := range p.Spec.Containers {
				featureGates, found := findFeatureGatesFlagInContainerArgs(c.Args)
				if !found {
					fmt.Printf("    %s/%s: No Feature Gates Found\n", p.Name, c.Name)
					continue
				}

				featureGateKVPairs := parseFeatureGatesValue(featureGates)
				fmt.Printf("    %s/%s:\n", p.Name, c.Name)

				for k, v := range featureGateKVPairs {
					fmt.Printf("        %s: %s\n", k, v)
				}
			}
		}
	}
}

// constructSystemComponentPodsMap はシステムコンポーネントごとにPodリストを構築する
func constructSystemComponentPodsMap(pods *corev1.PodList) map[string][]corev1.Pod {
	m := map[string][]corev1.Pod{
		mapKeyKubeAPIServer: {},
		mapKeyKubeScheduler: {},
	}

	for _, p := range pods.Items {
		if strings.Contains(p.Name, mapKeyKubeAPIServer) {
			m[mapKeyKubeAPIServer] = append(m[mapKeyKubeAPIServer], p)
		}
		if strings.Contains(p.Name, mapKeyKubeScheduler) {
			m[mapKeyKubeScheduler] = append(m[mapKeyKubeScheduler], p)
		}
	}
	return m
}

// findFeatureGatesFlagInContainerArgs はコンテナのコマンドライン引数からFeature Gatesの指定を検索してその設定値を返す
// もしFeatureGatesフラグが存在しなければ第二引数にfalseが入る
func findFeatureGatesFlagInContainerArgs(args []string) (string, bool) {
	value := ""
	ok := false

	for _, arg := range args {
		if strings.Contains(arg, "--feature-gates") {
			value = strings.Split(arg, "=")[1]
		}
	}

	return value, ok
}

func parseFeatureGatesValue(rawFeatureGates string) map[string]string {
	m := make(map[string]string)

	for _, rawKVPair := range strings.Split(rawFeatureGates, ",") {
		tmp := strings.Split(rawKVPair, "=")
		featureGateName := tmp[0]
		featureGateValue := tmp[1]
		m[featureGateName] = featureGateValue
	}

	return m
}
