package collector

import (
	"context"
	"log"
	"os"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Client struct {
	k8sClient *kubernetes.Clientset
}

func NewClient() (Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return Client{}, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return Client{}, err
	}

	return Client{
		k8sClient: clientset,
	}, nil
}

func (k Client) Watch(ctx context.Context, labelMap map[string]string, fn func(collectors []string)) {
	collectors := []string{}

	ns := os.Getenv("OTEL_NAMESPACE")
	opts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}
	pods, err := k.k8sClient.CoreV1().Pods(ns).List(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}
	for i := range pods.Items {
		collectors = append(collectors, pods.Items[i].Name)
	}
	fn(collectors)

	watcher, err := k.k8sClient.CoreV1().Pods(ns).Watch(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				for event := range watcher.ResultChan() {
					pod, ok := event.Object.(*v1.Pod)
					if !ok {
						log.Fatal(err)
					}

					switch event.Type {
					case watch.Added:
						collectors = append(collectors, pod.Name)

					case watch.Deleted:
						for i, _ := range collectors {
							if collectors[i] == pod.Name {
								collectors = append(collectors[:i], collectors[i+1:]...)
							}
						}
					}
					fn(collectors)
				}
			}
		}
	}()
}
