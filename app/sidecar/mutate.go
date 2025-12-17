package sidecar

import (
	"encoding/json"
	"fmt"
	"kube-sidecar/app"
	"kube-sidecar/z"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/klog/v2"
)

func init() {
	z.Register("99-app.sidecar", func(svc z.SvcKit, enr z.Enroll) z.Closed {
		api := z.RegSvc(svc, &MutateApi{Patcher: NewInjectorPatcher(svc)})
		enr(z.POST("mutate"), api.mutate) // 注册接口
		return nil
	})
}

type MutateApi struct {
	SvcKit  z.SvcKit   `svckit:"-"`
	Patcher PodPatcher `svckit:"-"`
}

func (aa *MutateApi) mutate(zrc *z.Ctx) bool {
	if err := app.PostJson(zrc.Request); err != nil {
		klog.Error(err.Error())
		writeErrorAdmissionReview(http.StatusBadRequest, err.Error(), zrc.Writer)
		return true
	}
	admReview, err := z.ReadBody(zrc.Request, &admissionv1.AdmissionReview{})
	if err != nil {
		klog.Errorf("Could not decode body: %v", err)
		writeErrorAdmissionReview(http.StatusInternalServerError, err.Error(), zrc.Writer)
		return true
	}
	req := admReview.Request

	klog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v UID=%v patchOperation=%v UserInfo=%v", //
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)
	if patchOperations, err := aa.process(zrc, req); err != nil {
		message := fmt.Sprintf("request for object '%s' with name '%s' in namespace '%s' denied: %v", //
			req.Kind.String(), req.Name, req.Namespace, err)
		klog.Error(message)
		writeDeniedAdmissionResponse(admReview, message, zrc.Writer)
	} else if patchBytes, err := json.Marshal(patchOperations); err != nil {
		message := fmt.Sprintf("request for object '%s' with name '%s' in namespace '%s' denied: %v", //
			req.Kind.String(), req.Name, req.Namespace, err)
		klog.Error(message)
		writeDeniedAdmissionResponse(admReview, message, zrc.Writer)
	} else {
		writeAllowedAdmissionReview(admReview, patchBytes, zrc.Writer)
	}

	// return zrc.JSON(&z.Result{Success: true, Data: "mutate"})
	return true
}

func (aa *MutateApi) process(zrc *z.Ctx, req *admissionv1.AdmissionRequest) ([]PatchOperation, error) {
	switch req.Operation {
	case admissionv1.Create:
		return aa.handleAdmissionCreate(zrc, req)
	case admissionv1.Update:
		return aa.handleAdmissionUpdate(zrc, req)
	case admissionv1.Delete:
		return aa.handleAdmissionDelete(zrc, req)
	default:
		return nil, fmt.Errorf("unhandled request operations type %s", req.Operation)
	}
}

func (aa *MutateApi) handleAdmissionCreate(zrc *z.Ctx, req *admissionv1.AdmissionRequest) ([]PatchOperation, error) {
	pod, err := unmarshalPod(req.Object.Raw)
	if err != nil {
		return nil, err
	}
	return aa.Patcher.PatchPodCreate(zrc.Ctx, req.Namespace, pod)
}

func (aa *MutateApi) handleAdmissionUpdate(zrc *z.Ctx, req *admissionv1.AdmissionRequest) ([]PatchOperation, error) {
	oldPod, err := unmarshalPod(req.OldObject.Raw)
	if err != nil {
		return nil, err
	}
	newPod, err := unmarshalPod(req.Object.Raw)
	if err != nil {
		return nil, err
	}
	return aa.Patcher.PatchPodUpdate(zrc.Ctx, req.Namespace, oldPod, newPod)
}

func (aa *MutateApi) handleAdmissionDelete(zrc *z.Ctx, req *admissionv1.AdmissionRequest) ([]PatchOperation, error) {
	pod, err := unmarshalPod(req.OldObject.Raw)
	if err != nil {
		return nil, err
	}
	return aa.Patcher.PatchPodDelete(zrc.Ctx, req.Namespace, pod)
}
