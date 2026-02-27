package sidecar

import (
	"encoding/json"
	"flag"
	"fmt"
	"k8skit/app/k8sc"
	"net/http"
	"strings"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var (
	C = struct {
		Sidecar struct {
			Config
			DB sqlx.DatabaseConfig `json:"database"`
		}
	}{}
)

type Config struct {
	Annotation string `json:"annotation"`
	DefaultKey string `json:"defaultKey"`
	ByDBConfig string `json:"byDbConfig"`
	ByDBFolder string `json:"byDbFolder"`
	ByDBAppEnv string `json:"byDbAppEnv"`
	ServerHost string `json:"serverHost"`
	InitcImage string `json:"initcImage"`
}

func init() {
	z.Config(&C)
	flag.StringVar(&C.Sidecar.Annotation, "sidecarAnnotation", "ksidecar/configmap", "injector annotation, namespace/configmap#attribute")
	flag.StringVar(&C.Sidecar.DefaultKey, "sidecarDefaultKey", "value.yml", "injector default configmap attribute name")
	flag.StringVar(&C.Sidecar.ByDBConfig, "sidecarByDBConfig", "ksidecar/db.config", "injector configuration, (app)(.json|yaml|prop|toml)(:version)(#0), db.config > container.name > labels[app]")
	flag.StringVar(&C.Sidecar.ByDBFolder, "sidecarByDBFolder", "ksidecar/db.folder", "injector configuration directory path")
	flag.StringVar(&C.Sidecar.ByDBAppEnv, "sidecarByDBAppEnv", "ksidecar/db.appenv", "run image environment, [dev, fat, uat, pro...]")
	flag.StringVar(&C.Sidecar.ServerHost, "sidecarServerHost", "http://ksidecar.default.svc", "injector server host")
	flag.StringVar(&C.Sidecar.InitcImage, "sidecarInitcImage", "suisrc/k8skit:1.3.15-wgetar", "init container archive image")

	flag.StringVar(&C.Sidecar.DB.Driver, "sidecarDriver", "mysql", "sqlx driver name")
	flag.StringVar(&C.Sidecar.DB.DataSource, "sidecarDatasource", "", "sqlx data source name")
	flag.StringVar(&C.Sidecar.DB.TablePrefix, "sidecarTablePrefix", "", "sqlx table prefix")

	z.Register("99-app.sidecar", func(zgg *z.Zgg) z.Closed {
		dsc, err := sqlx.ConnectDatabase(&C.Sidecar.DB)
		if err != nil {
			zgg.ServeStop(err.Error())
			return nil
		} else {
			// 链接成功， 打印链接信息
			dsn := C.Sidecar.DB.DataSource
			if idx := strings.Index(dsn, "@"); idx > 0 {
				usr := dsn[:idx]
				dsn = dsn[idx+1:]
				if idz := strings.Index(usr, ":"); idz > 0 {
					dsn = usr[:idz] + ":******@" + dsn
				}
			}
			z.Println("[database]: connect ok,", dsn)
		}
		// 注册服务
		api := z.RegSvc(zgg.SvcKit, &MutateApi{Patcher: &Patcher{
			Config:    C.Sidecar.Config,
			K8sClient: zgg.SvcKit.Get("k8sclient").(kubernetes.Interface),
			ConfRepo:  NewConfRepo(dsc),
		}})
		// router
		zgg.AddRouter(http.MethodPost+" mutate", api.mutate)
		zgg.AddRouter(http.MethodGet+" archive", api.archive)
		// z.POST("mutate", api.mutate, zgg) // 注册接口
		if dsc != nil {
			return func() { dsc.Close() }
		}
		return nil
	})
}

// Patcher Sidecar Injector patcher
type Patcher struct {
	Config    Config
	K8sClient kubernetes.Interface
	ConfRepo  *ConfRepo

	AllowAnnotationOverrides bool
	AllowLabelOverrides      bool
}

// MutateApi Sidecar Injector api
type MutateApi struct {
	SvcKit  z.SvcKit `svckit:"-"`
	Patcher *Patcher `svckit:"-"`
}

func (aa *MutateApi) mutate(zrc *z.Ctx) {
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
	// AdmissionReview
	// bts, _ := json.Marshal(admReview)
	// z.Println("AdmissionReview: ", string(bts))
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
