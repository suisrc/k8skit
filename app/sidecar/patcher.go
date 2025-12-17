package sidecar

import (
	"context"
	"kube-sidecar/z"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// PodPatcher Pod patching interface
type PodPatcher interface {
	PatchPodCreate(ctx context.Context, namespace string, pod corev1.Pod) ([]PatchOperation, error)
	PatchPodUpdate(ctx context.Context, namespace string, oldPod corev1.Pod, newPod corev1.Pod) ([]PatchOperation, error)
	PatchPodDelete(ctx context.Context, namespace string, pod corev1.Pod) ([]PatchOperation, error)
}

func NewInjectorPatcher(svc z.SvcKit) PodPatcher {
	config := svc.Srv().GetConfig()
	return &InjectorPatcher{
		K8sClient:        svc.Get("k8sclient").(kubernetes.Interface),
		InjectAnnotation: config.V("InjectAnnotation").(string),
		InjectDefaultKey: config.V("InjectDefaultKey").(string),
	}
}
