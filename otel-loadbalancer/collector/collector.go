package collector

import (
	"context"
	"log"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	watcherTimeout = 15 * time.Minute
)

var (
	ns = os.Getenv("OTEL_NAMESPACE")
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

func Get(ctx context.Context, LabelSelector map[string]string) ([]string, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	collectors := []string{}
	pods, err := clientset.CoreV1().Pods(os.Getenv("OTEL_NAMESPACE")).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(LabelSelector).String(),
	})
	if err != nil {
		return nil, err
	}
	for i := range pods.Items {
		pod := pods.Items[i]
		if pod.GetObjectMeta().GetDeletionTimestamp() == nil {
			collectors = append(collectors, pod.Name)
		}
	}

	return collectors, nil
}

func (k Client) Watch(ctx context.Context, labelMap map[string]string, fn func(collectors []string)) {
	collectors := []string{}

	opts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}
	pods, err := k.k8sClient.CoreV1().Pods(ns).List(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}
	for i := range pods.Items {
		pod := pods.Items[i]
		if pod.GetObjectMeta().GetDeletionTimestamp() == nil {
			collectors = append(collectors, pod.Name)
		}
	}
	fn(collectors)

	for {
		watcher, err := k.k8sClient.CoreV1().Pods(ns).Watch(ctx, opts)
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(watcherTimeout):
					return
				default:
					c := watcher.ResultChan()
					for event := range c {
						pod, ok := event.Object.(*v1.Pod)
						if !ok {
							log.Fatal(err)
						}

						switch event.Type {
						case watch.Added:
							collectors = append(collectors, pod.Name)

						case watch.Deleted:
							for i := range collectors {
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
}
