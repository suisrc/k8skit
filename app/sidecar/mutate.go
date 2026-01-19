package sidecar

import (
	"encoding/json"
	"flag"
	"fmt"
	"k8skit/app"
	"k8skit/app/repo"
	"net/http"

	"github.com/suisrc/zgg/z"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var (
	C = struct {
		InjectAnnotation string
		InjectDefaultKey string
		InjectConfigKind string
		InjectConfigPath string
		InjectServerHost string
	}{}
)

func init() {
	z.Config(&C)
	flag.StringVar(&C.InjectAnnotation, "injectAnnotation", "ksidecar/configmap", "injector annotation, namespace/configmap#attribute")
	flag.StringVar(&C.InjectDefaultKey, "injectDefaultKey", "value.yml", "injector default configmap attribute name")
	flag.StringVar(&C.InjectConfigKind, "injectConfigKind", "ksidecar/confkind", "injector configuration, [env](.json|yaml|prop|toml)(#0), #0 is containers offset from 0")
	flag.StringVar(&C.InjectConfigPath, "injectConfigPath", "ksidecar/confpath", "injector configuration directory path")
	flag.StringVar(&C.InjectServerHost, "injectServerHost", "http://ksidecar-injector.default.svc", "injector server host")

	z.Register("99-app.sidecar", func(zgg *z.Zgg) z.Closed {
		api := z.RegSvc(zgg.SvcKit, &MutateApi{Patcher: &Patcher{
			K8sClient:        zgg.SvcKit.Get("k8sclient").(kubernetes.Interface),
			ConfxRepository:  zgg.SvcKit.Get("repoconfx").(*repo.ConfxRepo),
			InjectAnnotation: C.InjectAnnotation,
			InjectDefaultKey: C.InjectDefaultKey,
			InjectConfigKind: C.InjectConfigKind,
			InjectConfigPath: C.InjectConfigPath,
			InjectServerHost: C.InjectServerHost,
		}})
		// router
		zgg.AddRouter(http.MethodPost+" mutate", api.mutate)
		zgg.AddRouter(http.MethodGet+" gitrepo", api.gitrepo)
		// z.POST("mutate", api.mutate, zgg) // 注册接口
		return nil
	})
}

// Patcher Sidecar Injector patcher
type Patcher struct {
	K8sClient                kubernetes.Interface
	ConfxRepository          *repo.ConfxRepo
	AllowAnnotationOverrides bool
	AllowLabelOverrides      bool
	InjectAnnotation         string
	InjectDefaultKey         string
	InjectConfigKind         string
	InjectConfigPath         string
	InjectServerHost         string
}

// MutateApi Sidecar Injector api
type MutateApi struct {
	SvcKit  z.SvcKit `svckit:"-"`
	Patcher *Patcher `svckit:"-"`
}

func (aa *MutateApi) mutate(zrc *z.Ctx) {
	if err := app.PostJson(zrc.Request); err != nil {
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

	z.Printf("AdmissionReview for Kind=%v, Namespace=%v Name=%v UID=%v patchOperation=%v UserInfo=%v", //
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
	// zrc.JSON(&z.Result{Success: true, Data: "mutate"})
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
