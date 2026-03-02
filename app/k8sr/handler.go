package k8s

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/suisrc/zgg/z"
	"go.yaml.in/yaml/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	_ "k8skit/app/k8sc"
)

func InitServe(zgg *z.Zgg) z.Closed {
	api := z.Inject(zgg.SvcKit, &K8sApi{})
	z.GET("k8s/v1/nss", api.nss, zgg)
	z.GET("k8s/v1/apps", api.apps, zgg)
	z.GET("k8s/v1/app", api.app, zgg)
	z.GET("k8s/v1/ings", api.ings, zgg)
	return nil

}

type K8sApi struct {
	K8sClient kubernetes.Interface `svckit:"k8sclient"`
}

func (api *K8sApi) nss(zrc *z.Ctx) {
	if ok := CheckToken(zrc); !ok {
		return
	}
	cli := api.K8sClient
	nss, err := cli.CoreV1().Namespaces().List(zrc.Ctx, metav1.ListOptions{})
	if err != nil {
		zrc.JERR(err, 500)
		return
	}
	names := []string{}
	for _, item := range nss.Items {
		if idx := slices.Index(C.K8sSync.ExcNs, item.Name); idx >= 0 {
			continue // 排除
		}
		names = append(names, item.Name)
	}
	zrc.JSON(&z.Result{Success: true, Data: names})
}
func (api *K8sApi) apps(zrc *z.Ctx) {
	if ok := CheckToken(zrc); !ok {
		return
	}
	qry := zrc.Request.URL.Query()
	ns := zrc.Request.URL.Query().Get("ns")
	if ns != "" {
		if idx := slices.Index(C.K8sSync.ExcNs, ns); idx >= 0 {
			zrc.JERR(fmt.Errorf("namespace excluded"), 403)
			return
		}
		// 获取指定命名空间下的应用列表
		cli := api.K8sClient
		apps, err := cli.AppsV1().Deployments(ns).List(zrc.Ctx, metav1.ListOptions{})
		if err != nil {
			zrc.JERR(err, 500)
			return
		}
		infos := make([]any, len(apps.Items))
		for i, app := range apps.Items {
			app.Kind = "Deployment"
			app.APIVersion = "apps/v1"
			infos[i] = api.toAnyMap(zrc, app)
		}
		if qry.Get("yaml") == "1" {
			str := &strings.Builder{}
			for _, item := range infos {
				fmt.Fprintf(str, "\n---\n%s", item.(map[string]any)["yaml"])
			}
			zrc.TEXT(str.String(), 200)
			return
		}
		zrc.JSON(&z.Result{Success: true, Data: infos})
		return
	}
	// 获取所有命名空间下的应用列表， 排除禁止同步的命名空间
	cli := api.K8sClient
	nss, err := cli.CoreV1().Namespaces().List(zrc.Ctx, metav1.ListOptions{})
	if err != nil {
		zrc.JERR(err, 500)
		return
	}
	infos := []any{}
	for _, ns := range nss.Items {
		if idx := slices.Index(C.K8sSync.ExcNs, ns.Name); idx >= 0 {
			continue // 排除
		}
		apps, err := cli.AppsV1().Deployments(ns.Name).List(zrc.Ctx, metav1.ListOptions{})
		if err != nil {
			zrc.JERR(err, 500)
			return
		}
		for _, app := range apps.Items {
			app.Kind = "Deployment"
			app.APIVersion = "apps/v1"
			infos = append(infos, api.toAnyMap(zrc, app))
		}
	}
	if qry.Get("yaml") == "1" {
		str := &strings.Builder{}
		for _, item := range infos {
			fmt.Fprintf(str, "\n---\n%s", item.(map[string]any)["yaml"])
		}
		zrc.TEXT(str.String(), 200)
		return
	}
	zrc.JSON(&z.Result{Success: true, Data: infos})
}

func (api *K8sApi) app(zrc *z.Ctx) {
	if ok := CheckToken(zrc); !ok {
		return
	}
	qry := zrc.Request.URL.Query()
	ns := qry.Get("ns")
	app := qry.Get("app")
	kind := qry.Get("kind")
	if ns == "" || app == "" || kind == "" {
		zrc.JERR(fmt.Errorf("no namespace or application or kind"), 400)
		return
	}
	cli := api.K8sClient
	switch kind {
	case "Deployment":
		app, err := cli.AppsV1().Deployments(ns).Get(zrc.Ctx, app, metav1.GetOptions{})
		if err != nil {
			zrc.JERR(err, 500)
			return
		}
		app.Kind = "Deployment"
		app.APIVersion = "apps/v1"
		item := api.toAnyMap(zrc, *app)
		if qry.Get("yaml") == "1" {
			zrc.TEXT((item.(map[string]any))["yaml"].(string), 200)
			return
		}
		zrc.JSON(&z.Result{Success: true, Data: item})
	case "StatefulSet":
		app, err := cli.AppsV1().StatefulSets(ns).Get(zrc.Ctx, app, metav1.GetOptions{})
		if err != nil {
			zrc.JERR(err, 500)
			return
		}
		app.Kind = "StatefulSet"
		app.APIVersion = "apps/v1"
		item := api.toAnyMap(zrc, *app)
		if qry.Get("yaml") == "1" {
			zrc.TEXT((item.(map[string]any))["yaml"].(string), 200)
			return
		}
		zrc.JSON(&z.Result{Success: true, Data: item})
	case "DaemonSet":
		app, err := cli.AppsV1().DaemonSets(ns).Get(zrc.Ctx, app, metav1.GetOptions{})
		if err != nil {
			zrc.JERR(err, 500)
			return
		}
		app.Kind = "DaemonSet"
		app.APIVersion = "apps/v1"
		item := api.toAnyMap(zrc, *app)
		if qry.Get("yaml") == "1" {
			zrc.TEXT((item.(map[string]any))["yaml"].(string), 200)
			return
		}
		zrc.JSON(&z.Result{Success: true, Data: item})
	default:
		zrc.JERR(fmt.Errorf("invalid kind"), 400)
	}
}

func (api *K8sApi) ings(zrc *z.Ctx) {
	if ok := CheckToken(zrc); !ok {
		return
	}
	qry := zrc.Request.URL.Query()
	ns := zrc.Request.URL.Query().Get("ns")
	if ns != "" {
		if idx := slices.Index(C.K8sSync.ExcNs, ns); idx >= 0 {
			zrc.JERR(fmt.Errorf("namespace excluded"), 403)
			return
		}
		// 获取指定命名空间下的应用列表
		cli := api.K8sClient
		ings, err := cli.NetworkingV1().Ingresses(ns).List(zrc.Ctx, metav1.ListOptions{})
		if err != nil {
			zrc.JERR(err, 500)
			return
		}
		infos := make([]any, len(ings.Items))
		for i, item := range ings.Items {
			item.Kind = "Ingress"
			item.APIVersion = "networking.k8s.io/v1"
			infos[i] = api.toAnyMap(zrc, item)
		}
		if qry.Get("yaml") == "1" {
			str := &strings.Builder{}
			for _, item := range infos {
				fmt.Fprintf(str, "\n---\n%s", item.(map[string]any)["yaml"])
			}
			zrc.TEXT(str.String(), 200)
			return
		}
		zrc.JSON(&z.Result{Success: true, Data: infos})
		return
	}
	// 获取所有命名空间下的应用列表， 排除禁止同步的命名空间
	cli := api.K8sClient
	nss, err := cli.CoreV1().Namespaces().List(zrc.Ctx, metav1.ListOptions{})
	if err != nil {
		zrc.JERR(err, 500)
		return
	}
	infos := []any{}
	for _, ns := range nss.Items {
		if idx := slices.Index(C.K8sSync.ExcNs, ns.Name); idx >= 0 {
			continue // 排除
		}
		ings, err := cli.NetworkingV1().Ingresses(ns.Name).List(zrc.Ctx, metav1.ListOptions{})
		if err != nil {
			zrc.JERR(err, 500)
		}
		for _, item := range ings.Items {
			item.Kind = "Ingress"
			item.APIVersion = "networking.k8s.io/v1"
			infos = append(infos, api.toAnyMap(zrc, item))
		}
	}
	if qry.Get("yaml") == "1" {
		str := &strings.Builder{}
		for _, item := range infos {
			fmt.Fprintf(str, "\n---\n%s", item.(map[string]any)["yaml"])
		}
		zrc.TEXT(str.String(), 200)
		return
	}
	zrc.JSON(&z.Result{Success: true, Data: infos})
}

// =================================================================================================

func (api *K8sApi) toAnyMap(zrc *z.Ctx, obj any) any {
	raw := map[string]any{}
	if bts, err := json.Marshal(obj); err != nil {
		return []any{}
	} else if err := json.Unmarshal(bts, &raw); err != nil {
		return []any{}
	}

	delete(raw, "status") // 删除状态字段
	if mate, ok := raw["metadata"].(map[string]any); ok {
		if anno, ok := mate["annotations"].(map[string]any); ok {
			delete(anno, "kubectl.kubernetes.io/last-applied-configuration")
			delete(anno, "deployment.kubernetes.io/revision")
			delete(anno, "statefulset.kubernetes.io/revision")
			delete(anno, "daemonset.kubernetes.io/revision")
		}
		// 删除扩展信息
		delete(mate, "uid")
		delete(mate, "resourceVersion")
		delete(mate, "generation")
		delete(mate, "creationTimestamp")
		delete(mate, "managedFields")
	}
	if spec, ok := raw["spec"].(map[string]any); ok {
		delete(spec, "clusterIP")
		delete(spec, "clusterIPs")
		delete(spec, "internalTrafficPolicy")
		delete(spec, "ipFamilies")
		delete(spec, "ipFamilyPolicy")
	}
	namespace := raw["metadata"].(map[string]any)["namespace"].(string)
	ado := map[string]any{}
	ado["kind"] = raw["kind"]
	ado["namespace"] = namespace
	ado["name"] = raw["metadata"].(map[string]any)["name"]
	bts, _ := yaml.Marshal([]any{raw})
	txt := string(bts)
	// -----------------------------------------------------------------------
	var label any // ["selector"].(map[string]any)["matchLabels"].(map[string]any)["app"]
	if vmap, _ := raw["spec"].(map[string]any); vmap == nil {
	} else if vmap, _ := vmap["selector"].(map[string]any); vmap == nil {
	} else if vmap, _ := vmap["matchLabels"].(map[string]any); vmap == nil {
	} else {
		label, _ = vmap["app"]
	}
	if label != nil {
		ado["label"] = label
		svcs, err := api.K8sClient.CoreV1().Services(namespace).List(zrc.Ctx, metav1.ListOptions{})
		if err == nil {
			for _, svc := range svcs.Items {
				if svc.Spec.Selector["app"] != label {
					continue
				}
				svc.Kind = "Service"
				svc.APIVersion = "v1"
				srv := map[string]any{}
				bts, _ := json.Marshal(svc)
				json.Unmarshal(bts, &srv)
				delete(srv, "status") // 删除状态字段
				if mate, ok := srv["metadata"].(map[string]any); ok {
					if anno, ok := mate["annotations"].(map[string]any); ok {
						delete(anno, "kubectl.kubernetes.io/last-applied-configuration")
					}
					// 删除扩展信息
					delete(mate, "uid")
					delete(mate, "resourceVersion")
					delete(mate, "generation")
					delete(mate, "creationTimestamp")
					delete(mate, "managedFields")
				}
				if spec, ok := srv["spec"].(map[string]any); ok {
					delete(spec, "clusterIP")
					delete(spec, "clusterIPs")
					delete(spec, "internalTrafficPolicy")
					delete(spec, "ipFamilies")
					delete(spec, "ipFamilyPolicy")
				}
				bts, _ = yaml.Marshal([]any{srv})
				txt = fmt.Sprintf("%s\n---\n", string(bts)) + txt
				//
				ado["service"] = svc.Name
				break
			}
		}
		containers := raw["spec"].(map[string]any)["template"].(map[string]any)["spec"].(map[string]any)["containers"].([]any)
		for _, ctn := range containers {
			ctn, _ := ctn.(map[string]any)
			if ctn["name"] == "sidecar" {
				continue
			}
			envs, _ := ctn["envFrom"].([]any)
			for _, env := range envs {
				env := env.(map[string]any)
				if ref, _ := env["configMapRef"].(map[string]any); ref != nil {
					name := ref["name"].(string)
					cm, err := api.K8sClient.CoreV1().ConfigMaps(namespace).Get(zrc.Ctx, name, metav1.GetOptions{})
					if err != nil {
						continue
					}
					cm.Kind = "ConfigMap"
					cm.APIVersion = "v1"
					vma := map[string]any{}
					bts, _ := json.Marshal(cm)
					json.Unmarshal(bts, &vma)
					delete(vma, "status") // 删除状态字段
					if mate, ok := vma["metadata"].(map[string]any); ok {
						if anno, ok := mate["annotations"].(map[string]any); ok {
							delete(anno, "kubectl.kubernetes.io/last-applied-configuration")
						}
						// 删除扩展信息
						delete(mate, "uid")
						delete(mate, "resourceVersion")
						delete(mate, "generation")
						delete(mate, "creationTimestamp")
						delete(mate, "managedFields")
					}
					if spec, ok := vma["spec"].(map[string]any); ok {
						delete(spec, "clusterIP")
						delete(spec, "clusterIPs")
						delete(spec, "internalTrafficPolicy")
						delete(spec, "ipFamilies")
						delete(spec, "ipFamilyPolicy")
					}
					bts, _ = yaml.Marshal([]any{vma})
					txt = fmt.Sprintf("%s\n---\n", string(bts)) + txt
					//
					venv, _ := ado["configmap"].(map[string]string)
					if venv == nil {
						venv = map[string]string{}
						ado["configmap"] = venv
					}
					maps.Copy(venv, cm.Data)
				} else if ref := env["secretRef"].(map[string]any); ref != nil {
					name := ref["name"].(string)
					cm, err := api.K8sClient.CoreV1().Secrets(namespace).Get(zrc.Ctx, name, metav1.GetOptions{})
					if err != nil {
						continue
					}
					cm.Kind = "Secret"
					cm.APIVersion = "v1"
					vma := map[string]any{}
					bts, _ := json.Marshal(cm)
					json.Unmarshal(bts, &vma)
					delete(vma, "status") // 删除状态字段
					if mate, ok := vma["metadata"].(map[string]any); ok {
						if anno, ok := mate["annotations"].(map[string]any); ok {
							delete(anno, "kubectl.kubernetes.io/last-applied-configuration")
						}
						// 删除扩展信息
						delete(mate, "uid")
						delete(mate, "resourceVersion")
						delete(mate, "generation")
						delete(mate, "creationTimestamp")
						delete(mate, "managedFields")
					}
					if spec, ok := vma["spec"].(map[string]any); ok {
						delete(spec, "clusterIP")
						delete(spec, "clusterIPs")
						delete(spec, "internalTrafficPolicy")
						delete(spec, "ipFamilies")
						delete(spec, "ipFamilyPolicy")
					}
					bts, _ = yaml.Marshal([]any{vma})
					txt = fmt.Sprintf("%s\n---\n", string(bts)) + txt
					//
					venv, _ := ado["secret"].(map[string]string)
					if venv == nil {
						venv = map[string]string{}
						ado["secret"] = venv
					}
					maps.Copy(venv, cm.StringData)
				}
			}

		}
	}
	// -----------------------------------------------------------------------
	ado["yaml"] = txt
	return ado
}
