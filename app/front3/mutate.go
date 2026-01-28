package front3

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/tlsx"
	admissionv1 "k8s.io/api/admission/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (aa *F3Serve) MutateTLS(dir string) (*tls.Config, error) {
	config := tlsx.TLSAutoConfig{}
	if crtBts, err := os.ReadFile(filepath.Join(dir, "ca.crt")); err != nil {
		return nil, err
	} else {
		config.CaCrtBts = crtBts
	}
	if keyBts, err := os.ReadFile(filepath.Join(dir, "ca.key")); err != nil {
		return nil, err
	} else {
		config.CaKeyBts = keyBts
	}
	cfg := &tls.Config{}
	cfg.GetCertificate = (&config).GetCertificate
	return cfg, nil
}

func (aa *F3Serve) Mutate(rw http.ResponseWriter, rr *http.Request) {
	if err := checkPostJson(rr); err != nil {
		z.Println(err.Error())
		writeErrorAdmissionReview(http.StatusBadRequest, err.Error(), rw)
		return
	}
	admReview, err := z.ReadBody(rr, &admissionv1.AdmissionReview{})
	if err != nil {
		z.Printf("Could not decode body: %v", err)
		writeErrorAdmissionReview(http.StatusInternalServerError, err.Error(), rw)
		return
	}
	// bts, _ := json.Marshal(admReview)
	// z.Println("AdmissionReview: ", string(bts))
	req := admReview.Request

	z.Printf("AdmissionReview for Kind=%v, Namespace=%v Name=%v UID=%v patchOperation=%v UserInfo=%v", //
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)

	if patchOperations, err := aa.mutateProcess(req); err != nil {
		message := fmt.Sprintf("request for object '%s' with name '%s' in namespace '%s' denied: %v", //
			req.Kind.String(), req.Name, req.Namespace, err)
		z.Println(message)
		writeDeniedAdmissionResponse(admReview, message, rw)
	} else if patchBytes, err := json.Marshal(patchOperations); err != nil {
		message := fmt.Sprintf("request for object '%s' with name '%s' in namespace '%s' denied: %v", //
			req.Kind.String(), req.Name, req.Namespace, err)
		z.Println(message)
		writeDeniedAdmissionResponse(admReview, message, rw)
	} else {
		writeAllowedAdmissionReview(admReview, patchBytes, rw)
	}
}

func (aa *F3Serve) mutateProcess(req *admissionv1.AdmissionRequest) ([]PatchOperation, error) {
	/*
		frontend/db.fronta: sso@fmes/iam-signin:v1.0.1 如果没有版本，不进行限制
		frontend/db.fronta.name: 登录系统
		frontend/db.fronta.rootdir: /
		frontend/db.frontv.tproot: /ROOT_PATH
		frontend/db.frontv.imagepath: /www/data
		frontend/db.frontv.indexs: /www=,/embed=index.htm
		frontend/db.frontv.cdnuse: 'true'
		------------------------------------------------------------------
		frontend/service: frontend:http/path # 默认不执行注入
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
	case admissionv1.Delete: // 删除, 对应服务逻辑删除
		var old netv1.Ingress
		if err := json.Unmarshal(req.OldObject.Raw, &old); err != nil {
			return nil, errors.New("error unmarshalling old object, " + err.Error())
		}
		if _, err := aa.mutateUpdateFronta(&old, nil); err != nil {
			return nil, err // 错误处理
		}
	default:
		return nil, fmt.Errorf("unhandled request operations type %s", req.Operation)
	}

	return patchs, nil
}

func (aa *F3Serve) mutateUpdateFronta(old *netv1.Ingress, ing *netv1.Ingress) (result *PatchOperation, reserr error) {
	if old != nil && len(old.GetAnnotations()) > 0 {
		// 处理旧数据内容，从数据库中删除应用
		if oldapp, _ := old.GetAnnotations()["frontend/db.fronta"]; oldapp != "" {
			// 对数据库进行配置, 确定增加或者删除应用
			if idx := strings.IndexByte(oldapp, '@'); idx > 0 {
				oldapp = oldapp[:idx] // 获取应用名
			}
			newapp := ""
			if ing != nil && len(ing.GetAnnotations()) > 0 {
				newapp = ing.GetAnnotations()["frontend/db.fronta"]
				if idx := strings.IndexByte(newapp, '@'); idx > 0 {
					newapp = newapp[:idx] // 获取应用名
				}
			}
			if oldapp != newapp {
				// 删除应用
				if err := aa.AppRepo.DelByApp(oldapp); err != nil {
					z.Println("[_mutate_]:", "get app info form database error,", err.Error())
				} else {
					z.Println("[_mutate_]:", "delete app from database,", oldapp)
				}
			} else {
				z.Println("[_mutate_]:", "app info no change [app] field,", oldapp)
			}
		}
	}
	if ing == nil || len(ing.GetAnnotations()) == 0 {
		if ing != nil && z.IsDebug() {
			z.Println("[_mutate_]:", ing.Namespace, "|", ing.Name, "no annotations")
		}
		return nil, nil // 没有新的配置
	}
	// 处理编排内容
	svc, _ := ing.GetAnnotations()["frontend/service"]
	if svc == "" {
		if z.IsDebug() {
			z.Println("[_mutate_]:", ing.Namespace, "|", ing.Name, "no frontend service")
		}
		return nil, nil // 没有 service 是无法处理域名的
	}
	patch, host2, err := aa.mutateFrontPath(ing, svc)
	if err != nil {
		return nil, err // 无法处理注入
	} else if patch != nil {
		result = patch // 保存处理的结果
	}
	// 处理数据内容, 忽略下面执行过程中的异常
	cfg, _ := ing.GetAnnotations()["frontend/db.fronta"]
	if cfg == "" {
		if z.IsDebug() {
			z.Println("[_mutate_]:", ing.Namespace, "|", ing.Name, "no frontend database")
		}
		return // 没有配置注解
	}
	// 数据库中增加或者删除应用
	app := cfg
	img := ""
	ver := ""
	if idx := strings.IndexByte(app, '@'); idx > 0 {
		img = app[idx+1:]
		app = app[:idx] // 获取应用名
	}
	if idx := strings.IndexByte(img, ':'); idx > 0 {
		ver = cfg[idx+1:]
		img = img[:idx] // 获取镜像
	}
	if app == "" {
		if z.IsDebug() {
			z.Println("[_mutate_]:", ing.Namespace, "|", ing.Name, "no frontend database app")
		}
		return
	}
	// 通过数据库获取应用信息， 包括已经删除的应用
	appInfo, err := aa.AppRepo.GetByAppWithDelete(app)
	if err != nil && err != sql.ErrNoRows {
		z.Println("[_mutate_]:", "get app info form database error,", err.Error())
		return // 查询数据库发生异常
	}
	sql_ := "updated=?, updater=?, deleted=0, disable=0, app=?, ver=?, domain=?, rootdir=?"
	args := []any{time.Now(), z.AppName, app, ver, host2[0], host2[1]}
	pre_ := "frontend/db.fronta."
	len_ := len(pre_)
	for anno, data := range ing.GetAnnotations() {
		if strings.HasPrefix(anno, pre_) {
			key := anno[len_:]
			switch data {
			case "true":
				sql_ += "," + key + "=1"
			case "false":
				sql_ += "," + key + "=0"
			default:
				sql_ += "," + key + "=?"
				args = append(args, data)
			}
		}
	}
	if appInfo.ID > 0 {
		args = append(args, appInfo.ID)
		_, err = aa.AppRepo.Database.Exec("UPDATE "+aa.AppRepo.TableName()+" SET "+sql_+" WHERE id=?", args...)
		if err != nil {
			z.Println("[_mutate_]:", "update app info into database error,", err.Error())
			return // 更新数据库发生异常
		}
	} else {
		sql_ += ", created=?, creater=?"
		args = append(args, time.Now(), z.AppName)
		ret, err := aa.AppRepo.Database.Exec("INSERT "+aa.AppRepo.TableName()+" SET "+sql_, args...)
		if err != nil {
			z.Println("[_mutate_]:", "insert app info into database error,", err.Error())
			return // 插入数据库发生异常
		}
		appInfo.ID, _ = ret.LastInsertId()
		args = append(args, appInfo.ID)
	}
	z.Println("[_mutate_]:", "update/insert app info into database,", sql_, z.ToStr(args))
	if ver == "" || img == "" {
		if z.IsDebug() {
			z.Println("[_mutate_]:", ing.Namespace, "|", ing.Name, "no frontend database version or image")
		}
		return
	}
	// 需要更新应用版本信息
	verInfo, err := aa.VerRepo.GetTop1ByAidAndVerWithDelete(appInfo.ID, ver)
	if err != nil && err != sql.ErrNoRows {
		z.Println("[_mutate_]:", "get app version info form database error,", err.Error())
		return // 获取数据库发生异常
	}
	sql_ = "updated=?, updater=?, deleted=0, disable=0, aid=?, ver=?, image=?"
	args = []any{time.Now(), z.AppName, appInfo.ID, ver, img}
	pre_ = "frontend/db.frontv."
	len_ = len(pre_)
	for anno, data := range ing.GetAnnotations() {
		if strings.HasPrefix(anno, pre_) {
			key := anno[len_:]
			switch data {
			case "true":
				sql_ += "," + key + "=1"
			case "false":
				sql_ += "," + key + "=0"
			default:
				sql_ += "," + key + "=?"
				args = append(args, data)
			}
		}
	}
	if verInfo.ID > 0 {
		args = append(args, verInfo.ID)
		_, err = aa.VerRepo.Database.Exec("UPDATE "+aa.VerRepo.TableName()+" SET "+sql_+" WHERE id=?", args...)
		if err != nil {
			z.Println("[_mutate_]:", "update app version info into database error,", err.Error())
			return // 更新数据库发生异常
		}
	} else {
		sql_ += ", created=?, creater=?"
		args = append(args, time.Now(), z.AppName)
		ret, err := aa.VerRepo.Database.Exec("INSERT "+aa.VerRepo.TableName()+" SET "+sql_, args...)
		if err != nil {
			z.Println("[_mutate_]:", "insert app version info into database error,", err.Error())
			return // 插入数据库发生异常
		}
		verInfo.ID, _ = ret.LastInsertId()
		args = append(args, verInfo.ID)
	}
	z.Println("[_mutate_]:", "update/insert app version info into database,", sql_, z.ToStr(args))
	return
}

func (aa *F3Serve) mutateFrontPath(ing *netv1.Ingress, svc string) (*PatchOperation, [2]string, error) {
	// 由于后面要处理服务和域名已经路径问题，所以要求 len(rules) == 1 必须成立
	if len(ing.Spec.Rules) == 0 {
		return nil, [2]string{}, errors.New("ingress no rules")
	} else if len(ing.Spec.Rules) > 1 {
		return nil, [2]string{}, errors.New("ingress rules more than one")
	}
	rule := ing.Spec.Rules[0]
	if rule.Host == "" {
		return nil, [2]string{}, errors.New("ingress no host")
	}
	serviceName := svc
	servicePort := "http"
	servicePath := "/"
	if idx := strings.IndexByte(serviceName, ':'); idx >= 0 {
		servicePort = serviceName[idx+1:]
		serviceName = serviceName[:idx]
	}
	if idx := strings.IndexByte(servicePort, '/'); idx >= 0 {
		servicePath = servicePort[idx:]
		servicePort = servicePort[:idx]
	}
	// 检查和注入 rules 信息
	if rule.HTTP == nil {
		rule.HTTP = &netv1.HTTPIngressRuleValue{}
	}
	if rule.HTTP.Paths == nil {
		rule.HTTP.Paths = []netv1.HTTPIngressPath{}
	}
	for _, path := range rule.HTTP.Paths {
		if path.Path == servicePath || path.Backend.Service != nil && path.Backend.Service.Name == serviceName {
			return nil, [2]string{rule.Host, path.Path}, nil // 前端服务已经存在
		}
	}
	// 增加一个节点, 作为前端服务的入口
	path := netv1.HTTPIngressPath{
		Path:     servicePath,
		PathType: z.Ptr(netv1.PathTypePrefix),
		Backend: netv1.IngressBackend{
			Service: &netv1.IngressServiceBackend{
				Name: serviceName,
				Port: netv1.ServiceBackendPort{},
			},
		},
	}
	if port, err := strconv.Atoi(servicePort); err == nil {
		path.Backend.Service.Port.Number = int32(port) // 端口号
	} else {
		path.Backend.Service.Port.Name = servicePort // 端点名称
	}
	rule.HTTP.Paths = append(rule.HTTP.Paths, path)
	return &PatchOperation{
		Op:    "add",
		Path:  "/spec/rules/0/http/paths/-",
		Value: path,
	}, [2]string{rule.Host, path.Path}, nil
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
