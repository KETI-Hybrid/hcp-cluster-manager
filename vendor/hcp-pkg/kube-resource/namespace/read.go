package namespace

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetNamespace(clientset *kubernetes.Clientset, namespace string) *corev1.Namespace {
	namespaceList, _ := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})

	for i := range namespaceList.Items {
		if namespaceList.Items[i].Name == namespace {
			return &namespaceList.Items[i]
		}
	}
	return nil
}
