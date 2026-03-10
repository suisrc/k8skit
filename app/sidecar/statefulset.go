// 为 statefulset 提供 node 选择器

package sidecar

import (
	"encoding/json"
	"fmt"
	"k8skit/app/k8sc"
	"net/http"
	"strings"

	"github.com/suisrc/zgg/z"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/klog/v2"
)

func (aa *MutateApi) statefulset(zrc *z.Ctx) {
	if err := k8sc.PostJson(zrc.Request); err != nil {
		klog.Error(err.Error())
		writeErrorAdmissionReview(http.StatusBadRequest, err.Error(), zrc.Writer)
		return
	}
	admReview, err := z.ReadBody(zrc.Request, &admissionv1.AdmissionReview{})
	if err != nil {
		klog.Errorf("Could not decode body: %v", err)
		writeErrorAdmissionReview(http.StatusInternalServerError, err.Error(), zrc.Writer)
		return
	}
	req := admReview.Request

	z.Printf("AR for statefulset Kind=%v, Namespace=%v Name=%v UID=%v patchOperation=%v UserInfo=%v", //
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)
	if patchOperations, err := aa.processStatefulset(zrc, req); err != nil {
		message := fmt.Sprintf("request for object '%s' with name '%s' in namespace '%s' denied: %v", //
			req.Kind.String(), req.Name, req.Namespace, err)
		klog.Error(message)
		writeDeniedAdmissionResponse(admReview, message, zrc.Writer)
	} else if len(patchOperations) == 0 {
		writeAllowedAdmissionReview(admReview, nil, zrc.Writer)
	} else if patchBytes, err := json.Marshal(patchOperations); err != nil {
		message := fmt.Sprintf("request for object '%s' with name '%s' in namespace '%s' denied: %v", //
			req.Kind.String(), req.Name, req.Namespace, err)
		klog.Error(message)
		writeDeniedAdmissionResponse(admReview, message, zrc.Writer)
	} else {
		writeAllowedAdmissionReview(admReview, patchBytes, zrc.Writer)
	}
	// zrc.JSON(&z.Result{Success: true, Data: "mutate"})
}

func (aa *MutateApi) processStatefulset(zrc *z.Ctx, req *admissionv1.AdmissionRequest) ([]PatchOperation, error) {
	switch req.Operation {
	case admissionv1.Create:
		return aa.handleStatefulsetCreate(zrc, req)
	case admissionv1.Update:
		return nil, nil
	case admissionv1.Delete:
		return nil, nil
	default:
		return nil, fmt.Errorf("unhandled request operations type %s", req.Operation)
	}
}

func (aa *MutateApi) handleStatefulsetCreate(_ *z.Ctx, req *admissionv1.AdmissionRequest) ([]PatchOperation, error) {
	pod, err := unmarshalPod(req.Object.Raw)
	if err != nil {
		return nil, err
	}
	patches := []PatchOperation{}
	// __pod_index__, __pod_name__
	podName, podIndex := pod.Name, "0"
	if idx := strings.LastIndex(podName, "-"); idx > 0 {
		podIndex = podName[idx+1:]
	}
	if nodeSelector, exist := pod.Annotations["statefulset.kubernetes.io/node-selector"]; exist {
		// 将 nodeSelector 添加到 pod 的 nodeSelector 中
		nodeSelector = strings.ReplaceAll(strings.ReplaceAll(nodeSelector, "__pod_index__", podIndex), "__pod_name__", podName)
		values := strings.SplitN(nodeSelector, "=", 2)
		valmap := map[string]string{values[0]: values[1]}
		if len(values) == 2 {
			if len(pod.Spec.NodeSelector) == 0 {
				pod.Spec.NodeSelector = valmap
				patches = append(patches, PatchOperation{Op: "add", Path: "/spec/nodeSelector", Value: valmap})
			} else {
				pod.Spec.NodeSelector[values[0]] = values[1]
				patches = append(patches, CreateObjectPatches(valmap, &pod.Spec.NodeSelector, "/spec/nodeSelector", true)...)
			}
		}
	}
	if nodeName, exist := pod.Annotations["statefulset.kubernetes.io/node-name"]; exist {
		// 将 nodeName 添加到 pod 的 nodeName 中
		nodeName = strings.ReplaceAll(strings.ReplaceAll(nodeName, "__pod_index__", podIndex), "__pod_name__", podName)
		pod.Spec.NodeName = nodeName
		patches = append(patches, PatchOperation{Op: "add", Path: "/spec/nodeName", Value: nodeName})
	}
	return patches, nil
}
