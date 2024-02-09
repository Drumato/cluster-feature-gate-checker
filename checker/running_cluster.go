package checker

import (
	"context"
	"flag"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	mapKeyKubeAPIServer         = "kube-apiserver"
	mapKeyKubeScheduler         = "kube-scheduler"
	mapKeyKubeControllerManager = "kube-controller-manager"
	mapKeyKubeProxy             = "kube-proxy"
)

type SystemComponentFeatureGateMartix map[string][]SystemComponentPod

type SystemComponentPod struct {
	Name       string
	Containers []SystemComponentContainer
}

type SystemComponentContainer struct {
	Name         string
	FeatureGates []FeatureGateConfig
}

type FeatureGateConfig struct {
	FeatureGateKey   string
	FeatureGateValue string
}

// CollectRunningClusterFeatureGates はKubernetesクラスタからシステムコンポーネントの情報を取ってきて、Feature Gateの設定を解析する
// 返り値となるMatrixは、 {"kube-apiserver": [{"feature-gate-001": "true"}, {}]}のようになる
func CollectRunningClusterFeatureGates(ctx context.Context, clientset *kubernetes.Clientset) (SystemComponentFeatureGateMartix, error) {
	m := make(map[string][]SystemComponentPod)

	pods, err := clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		return m, err
	}

	podMap := constructSystemComponentPodsMap(pods)

	for componentName, pods := range podMap {
		podConfigs := make([]SystemComponentPod, len(pods))

		for podIdx, p := range pods {
			containers := make([]SystemComponentContainer, len(p.Spec.Containers))
			for containerIdx, c := range p.Spec.Containers {
				containers[containerIdx].Name = c.Name
				featureGates, found := findFeatureGatesFlagInContainerArgs(c.Args)
				if !found {
					continue
				}

				featureGateKVPairs := parseFeatureGatesValue(featureGates)
				containers[containerIdx].FeatureGates = make([]FeatureGateConfig, len(featureGateKVPairs))

				featureGateCount := 0
				for k, v := range featureGateKVPairs {
					containers[containerIdx].FeatureGates[featureGateCount] = FeatureGateConfig{
						FeatureGateKey:   k,
						FeatureGateValue: v,
					}
				}
			}

			podConfigs[podIdx] = SystemComponentPod{
				Name:       p.Name,
				Containers: containers,
			}
		}

		m[componentName] = podConfigs
	}
	return m, nil
}

// constructSystemComponentPodsMap はシステムコンポーネントごとにPodリストを構築する
func constructSystemComponentPodsMap(pods *corev1.PodList) map[string][]corev1.Pod {
	m := map[string][]corev1.Pod{
		mapKeyKubeAPIServer:         {},
		mapKeyKubeScheduler:         {},
		mapKeyKubeControllerManager: {},
		mapKeyKubeProxy:             {},
	}

	for _, p := range pods.Items {
		if strings.Contains(p.Name, mapKeyKubeAPIServer) {
			m[mapKeyKubeAPIServer] = append(m[mapKeyKubeAPIServer], p)
		}
		if strings.Contains(p.Name, mapKeyKubeScheduler) {
			m[mapKeyKubeScheduler] = append(m[mapKeyKubeScheduler], p)
		}
		if strings.Contains(p.Name, mapKeyKubeControllerManager) {
			m[mapKeyKubeControllerManager] = append(m[mapKeyKubeControllerManager], p)
		}
		if strings.Contains(p.Name, mapKeyKubeProxy) {
			m[mapKeyKubeProxy] = append(m[mapKeyKubeProxy], p)
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

func setupFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("FlagSet", flag.ContinueOnError)

	return fs
}
