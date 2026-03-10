package k8s

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
	"go.yaml.in/yaml/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	_ "k8skit/app/k8sc"
)

func InitServe(zgg *z.Zgg) z.Closed {
	api := z.Inject(zgg.SvcKit, &K8sApi{})
	z.GET("k8s/sync/v1/nss", api.nss, zgg)
	z.GET("k8s/sync/v1/apps", api.apps, zgg)
	z.GET("k8s/sync/v1/app", api.app, zgg)
	z.GET("k8s/sync/v1/ings", api.ings, zgg)
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
	pageNo := 1
	if val := qry.Get("pageNo"); val == "" {
	} else if val, err := strconv.Atoi(val); err != nil {
	} else {
		pageNo = val
	}
	if pageNo < 1 {
		pageNo = 1
	}
	pageSize := 10
	if val := qry.Get("pageSize"); val == "" {
	} else if val, err := strconv.Atoi(val); err != nil {
	} else {
		pageSize = val
	}
	pageFirst := (pageNo - 1) * pageSize // 获取指定页的记录
	pageCount := pageSize
	//
	if ns != "" {
		if idx := slices.Index(C.K8sSync.ExcNs, ns); idx >= 0 {
			zrc.JERR(fmt.Errorf("namespace excluded"), 403)
			return
		}
		infos := []any{}
		api.apps_(zrc, ns, -1, pageFirst, pageCount, &infos)
		// 获取指定命名空间下的应用列表
		if zrc.IsAbort() {
			return
		}
		api.ResultArray(zrc, qry, infos)
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
	i := -1
	for _, ns := range nss.Items {
		if idx := slices.Index(C.K8sSync.ExcNs, ns.Name); idx >= 0 {
			continue // 排除
		}
		i = api.apps_(zrc, ns.Name, i, pageFirst, pageCount, &infos)
		if i < 0 {
			break
		}
	}
	if zrc.IsAbort() {
		return
	}
	api.ResultArray(zrc, qry, infos)
}

func (api *K8sApi) apps_(zrc *z.Ctx, ns string, oidx, pfst, psiz int, infos *[]any) int {
	cli := api.K8sClient
	{
		apps, err := cli.AppsV1().Deployments(ns).List(zrc.Ctx, metav1.ListOptions{})
		if err != nil {
			zrc.JERR(err, 500)
			return -2
		}
		for _, app := range apps.Items {
			oidx++
			if pfst > oidx {
				continue
			}
			if oidx-pfst >= psiz {
				return -3
			}
			app.Kind = "Deployment"
			app.APIVersion = "apps/v1"
			*infos = append(*infos, api.toAnyMap(zrc, app))
		}
	}
	if oidx-pfst < psiz {
		apps, err := cli.AppsV1().StatefulSets(ns).List(zrc.Ctx, metav1.ListOptions{})
		if err != nil {
			zrc.JERR(err, 500)
			return -2
		}
		for _, app := range apps.Items {
			oidx++
			if pfst > oidx {
				continue
			}
			if oidx-pfst >= psiz {
				return -3
			}
			app.Kind = "StatefulSet"
			app.APIVersion = "apps/v1"
			*infos = append(*infos, api.toAnyMap(zrc, app))
		}
	}
	if oidx-pfst < psiz {
		apps, err := cli.AppsV1().DaemonSets(ns).List(zrc.Ctx, metav1.ListOptions{})
		if err != nil {
			zrc.JERR(err, 500)
			return -2
		}
		for _, app := range apps.Items {
			oidx++
			if pfst > oidx {
				continue
			}
			if oidx-pfst >= psiz {
				return -3
			}
			app.Kind = "DaemonSet"
			app.APIVersion = "apps/v1"
			*infos = append(*infos, api.toAnyMap(zrc, app))
		}
	}
	return oidx
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
		api.ResultOne(zrc, qry, item)
	case "StatefulSet":
		app, err := cli.AppsV1().StatefulSets(ns).Get(zrc.Ctx, app, metav1.GetOptions{})
		if err != nil {
			zrc.JERR(err, 500)
			return
		}
		app.Kind = "StatefulSet"
		app.APIVersion = "apps/v1"
		item := api.toAnyMap(zrc, *app)
		api.ResultOne(zrc, qry, item)
	case "DaemonSet":
		app, err := cli.AppsV1().DaemonSets(ns).Get(zrc.Ctx, app, metav1.GetOptions{})
		if err != nil {
			zrc.JERR(err, 500)
			return
		}
		app.Kind = "DaemonSet"
		app.APIVersion = "apps/v1"
		item := api.toAnyMap(zrc, *app)
		api.ResultOne(zrc, qry, item)
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
	pageNo := 1
	if val := qry.Get("pageNo"); val == "" {
	} else if val, err := strconv.Atoi(val); err != nil {
	} else {
		pageNo = val
	}
	if pageNo < 1 {
		pageNo = 1
	}
	pageSize := 10
	if val := qry.Get("pageSize"); val == "" {
	} else if val, err := strconv.Atoi(val); err != nil {
	} else {
		pageSize = val
	}
	pageFirst := (pageNo - 1) * pageSize // 获取指定页的记录
	pageCount := pageSize
	//
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
		infos := []any{}
		for i, item := range ings.Items {
			if pageFirst > i {
				continue
			}
			if i-pageFirst >= pageCount {
				break
			}
			item.Kind = "Ingress"
			item.APIVersion = "networking.k8s.io/v1"
			infos = append(infos, api.toAnyMap(zrc, item))
		}
		api.ResultArray(zrc, qry, infos)
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
	i := -1
	for _, ns := range nss.Items {
		if idx := slices.Index(C.K8sSync.ExcNs, ns.Name); idx >= 0 {
			continue // 排除
		}
		ings, err := cli.NetworkingV1().Ingresses(ns.Name).List(zrc.Ctx, metav1.ListOptions{})
		if err != nil {
			zrc.JERR(err, 500)
			return
		}
		for _, item := range ings.Items {
			i++
			if pageFirst > i {
				continue
			}
			if i-pageFirst >= pageCount {
				break
			}
			item.Kind = "Ingress"
			item.APIVersion = "networking.k8s.io/v1"
			infos = append(infos, api.toAnyMap(zrc, item))
		}
	}
	api.ResultArray(zrc, qry, infos)
}

// =================================================================================================

func (api *K8sApi) toAnyMap(zrc *z.Ctx, obj any) any {
	raw := map[string]any{}
	if bts, err := json.Marshal(obj); err != nil {
		return []any{}
	} else if err := json.Unmarshal(bts, &raw); err != nil {
		return []any{}
	}
	api.ClearExInfo(raw)
	namespace := raw["metadata"].(map[string]any)["namespace"].(string)
	ado := map[string]any{}
	ado["kind"] = raw["kind"]
	ado["namespace"] = namespace
	ado["name"] = raw["metadata"].(map[string]any)["name"]
	bts, _ := yaml.Marshal(raw)
	yamlTxt := string(bts)
	jsonArr := []any{raw}
	// -----------------------------------------------------------------------
	var label any // ["selector"].(map[string]any)["matchLabels"].(map[string]any)["app"]
	labelkey := "app"
	if vmap, _ := raw["spec"].(map[string]any); vmap == nil {
	} else if vmap, _ := vmap["selector"].(map[string]any); vmap == nil {
	} else if vmap, _ := vmap["matchLabels"].(map[string]any); vmap == nil {
	} else {
		label, _ = vmap[labelkey]
		if label == nil {
			// 尝试二次获取， 如果还是失败就放弃
			labelkey = "app.kubernetes.io/name"
			label, _ = vmap[labelkey]
		}
	}
	if label != nil {
		ado["label"] = label
		svcs, ok := zrc.Caches["k8s-services-cache"].(*corev1.ServiceList)
		if !ok {
			var err error
			svcs, err = api.K8sClient.CoreV1().Services(namespace).List(zrc.Ctx, metav1.ListOptions{})
			if err != nil {
				svcs = &corev1.ServiceList{}
			}
			zrc.Caches["k8s-services-cache"] = svcs
		}
		for _, svc := range svcs.Items {
			if svc.Spec.Selector[labelkey] != label {
				continue
			}
			svc.Kind = "Service"
			svc.APIVersion = "v1"
			vma := map[string]any{}
			bts, _ := json.Marshal(svc)
			json.Unmarshal(bts, &vma)
			api.ClearExInfo(vma)
			jsonArr = append(jsonArr, vma)
			bts, _ = yaml.Marshal(vma)
			yamlTxt = fmt.Sprintf("%s\n---\n", string(bts)) + yamlTxt
			// 这里可能会存在多个 service 情况， 由于覆盖情况， ado["service"] 只取最后一个
			ado["service"] = svc.Name
			// break
		}
		// containers := raw["spec"].(map[string]any)["template"].(map[string]any)["spec"].(map[string]any)["containers"].([]any)
		containers, _ := zc.MapKey(raw, "spec.template.spec.containers").([]any)
		// configmap & secret
		for _, ctn := range containers {
			ctn, _ := ctn.(map[string]any)
			if ctn["name"] == "sidecar" {
				continue
			}
			envs, _ := ctn["envFrom"].([]any)
			for _, env := range envs {
				env := env.(map[string]any)
				if ref, _ := env["configMapRef"].(map[string]any); ref != nil {
					name, _ := ref["name"].(string)
					if name == "" {
						continue
					}
					cm, err := api.K8sClient.CoreV1().ConfigMaps(namespace).Get(zrc.Ctx, name, metav1.GetOptions{})
					if err != nil {
						continue
					}
					cm.Kind = "ConfigMap"
					cm.APIVersion = "v1"
					for k, v := range cm.Data {
						// cm.Data[k] = strings.TrimSpace(v)
						if strings.ContainsRune(v, '\n') {
							cm.Data[k] = FormatYamlString(v)
						}
					}
					vma := map[string]any{}
					bts, _ := json.Marshal(cm)
					json.Unmarshal(bts, &vma)
					api.ClearExInfo(vma)
					jsonArr = append(jsonArr, vma)
					bts, _ = yaml.Marshal(vma)
					yamlTxt = fmt.Sprintf("%s\n---\n", string(bts)) + yamlTxt
					//
					venv, _ := ado["configmap"].(map[string]string)
					if venv == nil {
						venv = map[string]string{}
						ado["configmap"] = venv
					}
					maps.Copy(venv, cm.Data)
				} else if ref, _ := env["secretRef"].(map[string]any); ref != nil {
					name, _ := ref["name"].(string)
					if name == "" {
						continue
					}
					cm, err := api.K8sClient.CoreV1().Secrets(namespace).Get(zrc.Ctx, name, metav1.GetOptions{})
					if err != nil {
						continue
					}
					cm.Kind = "Secret"
					cm.APIVersion = "v1"
					vma := map[string]any{}
					bts, _ := json.Marshal(cm)
					json.Unmarshal(bts, &vma)
					api.ClearExInfo(vma)
					jsonArr = append(jsonArr, vma)
					bts, _ = yaml.Marshal(vma)
					yamlTxt = fmt.Sprintf("%s\n---\n", string(bts)) + yamlTxt
					//
					venv, _ := ado["secret"].(map[string]string)
					if venv == nil {
						venv = map[string]string{}
						ado["secret"] = venv
					}
					maps.Copy(venv, cm.StringData)
				}
			}
			// volumes
			// volumes, _ := raw["spec"].(map[string]any)["template"].(map[string]any)["spec"].(map[string]any)["volumes"].([]any)
			volumes, _ := zc.MapKey(raw, "spec.template.spec.volumes").([]any)
			for _, vol := range volumes {
				vol := vol.(map[string]any)
				if ref, _ := vol["configMap"].(map[string]any); ref != nil {
					name := ref["name"].(string)
					if name == "" {
						continue
					}
					cm, err := api.K8sClient.CoreV1().ConfigMaps(namespace).Get(zrc.Ctx, name, metav1.GetOptions{})
					if err != nil {
						continue
					}
					cm.Kind = "ConfigMap"
					cm.APIVersion = "v1"
					for k, v := range cm.Data {
						// cm.Data[k] = strings.TrimSpace(v)
						if strings.ContainsRune(v, '\n') {
							cm.Data[k] = FormatYamlString(v)
						}
					}
					vma := map[string]any{}
					bts, _ := json.Marshal(cm)
					json.Unmarshal(bts, &vma)
					api.ClearExInfo(vma)
					jsonArr = append(jsonArr, vma)
					bts, _ = yaml.Marshal(vma)
					yamlTxt = fmt.Sprintf("%s\n---\n", string(bts)) + yamlTxt
					//
					venv, _ := ado["configmap_volume"].(map[string]string)
					if venv == nil {
						venv = map[string]string{}
						ado["configmap_volume"] = venv
					}
					maps.Copy(venv, cm.Data)
				} else if ref, _ := vol["secret"].(map[string]any); ref != nil {
					name, _ := ref["name"].(string)
					if name == "" {
						continue
					}
					cm, err := api.K8sClient.CoreV1().Secrets(namespace).Get(zrc.Ctx, name, metav1.GetOptions{})
					if err != nil {
						continue
					}
					cm.Kind = "Secret"
					cm.APIVersion = "v1"
					vma := map[string]any{}
					bts, _ := json.Marshal(cm)
					json.Unmarshal(bts, &vma)
					api.ClearExInfo(vma)
					jsonArr = append(jsonArr, vma)
					bts, _ = yaml.Marshal(vma)
					yamlTxt = fmt.Sprintf("%s\n---\n", string(bts)) + yamlTxt
					//
					venv, _ := ado["secret_volume"].(map[string]string)
					if venv == nil {
						venv = map[string]string{}
						ado["secret_volume"] = venv
					}
					maps.Copy(venv, cm.StringData)
				}
			}
		}
	}
	// -----------------------------------------------------------------------
	ado["yaml"] = yamlTxt
	ado["json"] = jsonArr
	return ado
}

func (api *K8sApi) ResultOne(zrc *z.Ctx, qry url.Values, rst any) {
	if qry.Get("yaml") == "1" {
		zrc.TEXT((rst.(map[string]any))["yaml"].(string), 200)
	} else if qry.Get("json") == "1" {
		arr := rst.(map[string]any)["json"].([]any)
		zrc.JSON(&z.Result{Success: true, Data: arr, Total: z.Ptr(len(arr))})
	} else {
		zrc.JSON(&z.Result{Success: true, Data: rst})
	}
}

func (api *K8sApi) ResultArray(zrc *z.Ctx, qry url.Values, rst []any) {
	if qry.Get("yaml") == "1" {
		str := &strings.Builder{}
		for _, item := range rst {
			fmt.Fprintf(str, "\n---\n%s", item.(map[string]any)["yaml"])
		}
		zrc.TEXT(str.String(), 200)
	} else if qry.Get("json") == "1" {
		arr := []any{}
		for _, i1 := range rst {
			for _, i2 := range i1.(map[string]any)["json"].([]any) {
				arr = append(arr, i2)
			}
		}
		zrc.JSON(&z.Result{Success: true, Data: arr, Total: z.Ptr(len(arr))})
	} else {
		zrc.JSON(&z.Result{Success: true, Data: rst})
	}
}

func (api *K8sApi) ClearExInfo(raw map[string]any) {
	delete(raw, "status") // 删除状态字段
	if mate, ok := raw["metadata"].(map[string]any); ok {
		if anno, ok := mate["annotations"].(map[string]any); ok {
			for k := range anno {
				if strings.HasPrefix(k, "deployment.kubernetes.io/") {
					delete(anno, k)
				} else if strings.HasPrefix(k, "kubectl.kubernetes.io/") {
					delete(anno, k)
				} else if strings.HasPrefix(k, "kubernetes.io/") {
					delete(anno, k)
				} else if strings.HasPrefix(k, "field.cattle.io/") {
					delete(anno, k)
				} else if strings.HasPrefix(k, "cattle.io/") {
					delete(anno, k)
				}
			}
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
		if temp, ok := spec["template"].(map[string]any); ok {
			if meta, ok := temp["metadata"].(map[string]any); ok {
				if anno, ok := meta["annotations"].(map[string]any); ok {
					for k := range anno {
						if strings.HasPrefix(k, "kubectl.kubernetes.io/") {
							delete(anno, k)
						} else if strings.HasPrefix(k, "kubernetes.io/") {
							delete(anno, k)
						} else if strings.HasPrefix(k, "field.cattle.io/") {
							delete(anno, k)
						} else if strings.HasPrefix(k, "cattle.io/") {
							delete(anno, k)
						}
					}
				}
			}
		}
	}
}

func FormatYamlString(str string) string {
	sbr := strings.Builder{}
	for line := range strings.SplitSeq(str, "\n") {
		sbr.WriteString(strings.TrimRightFunc(line, unicode.IsSpace))
		sbr.WriteRune('\n')
	}
	return strings.TrimRightFunc(sbr.String(), unicode.IsSpace)
}
