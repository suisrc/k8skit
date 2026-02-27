package front3

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/suisrc/zgg/z"
	admissionv1 "k8s.io/api/admission/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (aa *Serve) Mutate(rw http.ResponseWriter, rr *http.Request) {
	if err := checkPostJson(rr); err != nil {
		z.Println("[_mutate_]:", err.Error())
		writeErrorAdmissionReview(http.StatusBadRequest, err.Error(), rw)
		return
	}
	admReview, err := z.ReadBody(rr, &admissionv1.AdmissionReview{})
	if err != nil {
		z.Printf("[_mutate_]: Could not decode body: %v", err)
		writeErrorAdmissionReview(http.StatusInternalServerError, err.Error(), rw)
		return
	}
	// bts, _ := json.Marshal(admReview)
	// z.Println("[_mutate_]: AdmissionReview: ", string(bts))
	req := admReview.Request

	z.Printf("[_mutate_]: AdmissionReview for Kind=%v, Namespace=%v Name=%v UID=%v patchOperation=%v UserInfo=%v", //
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)

	if patchOperations, err := aa.mutateProcess(req); err != nil {
		message := fmt.Sprintf("request for object '%s' with name '%s' in namespace '%s' denied: %v", //
			req.Kind.String(), req.Name, req.Namespace, err)
		z.Println("[_mutate_]:", message)
		writeDeniedAdmissionResponse(admReview, message, rw)
	} else if /*len(patchOperations) == 0*/ patchOperations == nil {
		writeAllowedAdmissionReview(admReview, nil, rw)
	} else if patchBytes, err := json.Marshal(patchOperations); err != nil {
		message := fmt.Sprintf("request for object '%s' with name '%s' in namespace '%s' denied: %v", //
			req.Kind.String(), req.Name, req.Namespace, err)
		z.Println("[_mutate_]:", message)
		writeDeniedAdmissionResponse(admReview, message, rw)
	} else {
		writeAllowedAdmissionReview(admReview, patchBytes, rw)
	}
}

func (aa *Serve) mutateProcess(req *admissionv1.AdmissionRequest) ([]PatchOperation, error) {
	/*
		frontend/db.fronta: sso@fmes/iam-signin:v1.0.1 如果没有版本，不进行限制
		frontend/db.fronta.name: 登录系统
		frontend/db.fronta.rootdir: /
		frontend/db.frontv.tproot: /ROOT_PATH
		frontend/db.frontv.imagepath: /www/data
		frontend/db.frontv.indexs: /www=,/embed=index.htm
		frontend/db.frontv.cdnuse: 'true'
		------------------------------------------------------------------
		frontend/service: frontend:http/path # 如果不存在，不执行注入
		http:
		  paths:
		    - backend:
		        service:
		          name: frontend
		          port:
		            name: http
		      pathType: Prefix
		      path: /
	*/
	// 处理
	patchs := []PatchOperation{}
	switch req.Operation {
	case admissionv1.Create: // 创建, 更新或者创建应用
		var ing netv1.Ingress
		if err := json.Unmarshal(req.Object.Raw, &ing); err != nil {
			return nil, errors.New("error unmarshalling new object, " + err.Error())
		}
		if patch, err := aa.mutateUpdateFronta(nil, &ing); err != nil {
			return nil, err // 错误处理
		} else if patch != nil {
			patchs = append(patchs, *patch)
		}
		aa.mutateLogIngress(nil, &ing, req.Object.Raw)
	case admissionv1.Update: // 更新, 更新应用版本信息
		var old netv1.Ingress
		if err := json.Unmarshal(req.OldObject.Raw, &old); err != nil {
			return nil, errors.New("error unmarshalling old object, " + err.Error())
		}
		var ing netv1.Ingress
		if err := json.Unmarshal(req.Object.Raw, &ing); err != nil {
			return nil, errors.New("error unmarshalling new object, " + err.Error())
		}
		if patch, err := aa.mutateUpdateFronta(&old, &ing); err != nil {
			return nil, err // 错误处理
		} else if patch != nil {
			patchs = append(patchs, *patch)
		}
		aa.mutateLogIngress(&old, &ing, req.Object.Raw)
	case admissionv1.Delete: // 删除, 对应服务逻辑删除
		var old netv1.Ingress
		if err := json.Unmarshal(req.OldObject.Raw, &old); err != nil {
			return nil, errors.New("error unmarshalling old object, " + err.Error())
		}
		if _, err := aa.mutateUpdateFronta(&old, nil); err != nil {
			return nil, err // 错误处理
		}
		aa.mutateLogIngress(&old, nil, nil)
	default:
		return nil, fmt.Errorf("unhandled request operations type %s", req.Operation)
	}

	return patchs, nil
}

// =========================================================================================================================

// PatchOperation JsonPatch struct http://jsonpatch.com/
type PatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

func getIngressHosts(ing *netv1.Ingress) []string {
	hosts := []string{}
	for _, rule := range ing.Spec.Rules {
		host := rule.Host
		if host == "" {
			host = "*"
		}
		hosts = append(hosts, host)
	}
	return hosts
}

func checkPostJson(req *http.Request) error {
	if req.Method != http.MethodPost {
		return fmt.Errorf("wrong http verb. got %s", req.Method)
	}
	if req.Body == nil {
		return errors.New("empty body")
	}
	contentType := req.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return fmt.Errorf("wrong content type. expected 'application/json', got: '%s'", contentType)
	}
	return nil
}

func baseAdmissionReview() *admissionv1.AdmissionReview {
	gvk := admissionv1.SchemeGroupVersion.WithKind("AdmissionReview")
	return &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
	}
}

func admissionResponse(httpErrorCode int, message string) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Result: &metav1.Status{
			Code:    int32(httpErrorCode),
			Message: message,
		},
	}
}

func errorAdmissionReview(httpErrorCode int, message string) *admissionv1.AdmissionReview {
	r := baseAdmissionReview()
	r.Response = admissionResponse(httpErrorCode, message)
	return r
}

func writeAdmissionReview(r *admissionv1.AdmissionReview, res http.ResponseWriter) {
	resp, err := json.Marshal(r)
	if err != nil {
		z.Printf("Error marshalling decision: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = res.Write(resp)
	if err != nil {
		z.Printf("Error writing response: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func writeErrorAdmissionReview(status int, message string, res http.ResponseWriter) {
	admResp := errorAdmissionReview(status, message)
	writeAdmissionReview(admResp, res)
}

func writeDeniedAdmissionResponse(ar *admissionv1.AdmissionReview, message string, res http.ResponseWriter) {
	ar.Response = admissionResponse(http.StatusForbidden, message)
	ar.Response.UID = ar.Request.UID
	writeAdmissionReview(ar, res)
}

func writeAllowedAdmissionReview(ar *admissionv1.AdmissionReview, patch []byte, res http.ResponseWriter) {
	ar.Response = admissionResponse(http.StatusOK, "")
	ar.Response.Allowed = true
	ar.Response.UID = ar.Request.UID
	if patch != nil {
		pt := admissionv1.PatchTypeJSONPatch
		ar.Response.Patch = patch
		ar.Response.PatchType = &pt
	}
	writeAdmissionReview(ar, res)
}
