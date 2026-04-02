package front3

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/suisrc/zgg/z"
	netv1 "k8s.io/api/networking/v1"
)

// frontend/db.fronta=app[:version]
func (aa *Serve) mutateUpdateFronta(old *netv1.Ingress, ing *netv1.Ingress) (result *PatchOperation, reserr error) {
	// 处理前端应用
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
					z.Logn("[_mutate_]:", "get appinfo form database error,", err.Error())
				} else {
					z.Logn("[_mutate_]:", "delete app from database,", oldapp)
				}
			} else {
				z.Logn("[_mutate_]:", "appinfo field [app] no change,", oldapp)
			}
		}
	}
	if ing == nil || len(ing.GetAnnotations()) == 0 {
		if ing != nil && z.IsDebug() {
			z.Logn("[_mutate_]:", ing.Namespace, "|", ing.Name, "no annotations")
		}
		return nil, nil // 没有新的配置
	}
	// 处理编排内容=================================================================
	svc, _ := ing.GetAnnotations()["frontend/service"]
	if svc == "" {
		if z.IsDebug() {
			z.Logn("[_mutate_]:", ing.Namespace, "|", ing.Name, "no frontend service")
		}
		return nil, nil // 没有 service 是无法处理域名的
	}
	// host2[0] -> domain, host2[1] -> rootdir
	patch, host2, err := aa.mutateFrontPath(ing, svc)
	if err != nil {
		return nil, err // 无法处理注入
	} else if patch != nil {
		result = patch // 保存处理的结果
	}
	// 处理数据内容, 忽略下面执行过程中的异常
	app, _ := ing.GetAnnotations()["frontend/db.fronta"]
	if app == "" {
		if z.IsDebug() {
			z.Logn("[_mutate_]:", ing.Namespace, "|", ing.Name, "no frontend database")
		}
		return // 没有配置注解
	}
	// 数据库中增加或者删除应用
	ver := ""
	if idx := strings.IndexByte(app, ':'); idx > 0 {
		ver = app[idx+1:]
		app = app[:idx] // 重置应用名
	}
	if app == "" {
		if z.IsDebug() {
			z.Logn("[_mutate_]:", ing.Namespace, "|", ing.Name, "no frontend database app")
		}
		return
	}
	// 通过数据库获取应用信息， 包括已经删除的应用
	appInfo, err := aa.AppRepo.GetByAppWithDelete(app)
	if err != nil && err != sql.ErrNoRows {
		z.Logn("[_mutate_]:", "get appinfo form database error,", err.Error())
		return // 查询数据库发生异常
	}
	if rpath, _ := ing.GetAnnotations()["frontend/db.fronta.rootdir"]; rpath != "" {
		host2[1] = strings.TrimSpace(rpath) // 特殊情况下， 需要覆盖默认的根目录
	}
	// 确保在 appInfo 不存在的使用，也可以得到一个有效的 vpp 名称
	appInfo.App.String = app
	// host2[0] -> domain, 域名是不允许覆盖的
	err = aa.AppRepo.ModifyByInfo(appInfo, app, ver, host2[0], host2[1], ing.GetAnnotations())
	if err != nil {
		z.Logn("[_mutate_]:", "update appinfo into database error,", err.Error())
		return // 更新数据库发生异常
	}
	// 更新扩展应用信息，更新过程中忽略错误信息
	if str := ing.GetAnnotations()["frontend/db.fronts"]; strings.TrimSpace(str) != "" {
		infos := []map[string]string{}
		if err := json.Unmarshal([]byte(str), &infos); err != nil {
			z.Logn("[_mutate_]:", "unmarshal db.fronts error,", err.Error())
		} else if err := aa.VerRepo.UpdateByFrontsMap(infos); err != nil {
			z.Logn("[_mutate_]:", "update db.fronts into database error,", err.Error())
		}
	}
	// 处理 version 相关信息
	if ver == "" {
		// 通过 "frontend/db.frontv.ver" 获取版本
		ver = ing.GetAnnotations()["frontend/db.frontv.ver"]
	}
	// 通过 "frontend/db.frontv.image" 获取镜像
	img := ing.GetAnnotations()["frontend/db.frontv.image"]
	if img != "" {
		if strings.HasPrefix(img, "git+") || strings.HasPrefix(img, "https://") || strings.HasPrefix(img, "http://") {
			if idx := strings.LastIndexByte(img, '#'); idx > 0 && ver == "" {
				ver = img[idx+1:] // 截取版本
			} else if idx < 0 && ver != "" {
				img += "#" + ver // 添加版本
			}
		} else {
			if idx := strings.LastIndexByte(img, ':'); idx > 0 && ver == "" {
				ver = img[idx+1:] // 截取版本
			} else if idx < 0 && ver != "" {
				img += ":" + ver // 添加版本
			}
		}
	}
	if ver == "" || img == "" {
		if z.IsDebug() {
			z.Logn("[_mutate_]:", ing.Namespace, "|", ing.Name, "no frontend database version or image")
		}
		return
	}
	// 需要更新应用版本信息
	vpp := appInfo.GetVppName() // 获取最新的 vpp 名称， 注意，修改了 vpp， 可以导致之前的应用版本不可使用
	verInfo, err := aa.VerRepo.GetTop1ByVppAndVerWithDelete(vpp, ver)
	if err != nil && err != sql.ErrNoRows {
		z.Logn("[_mutate_]:", "get app version info form database error,", err.Error())
		return // 获取数据库发生异常
	}
	err = aa.VerRepo.ModifyByInfo(verInfo, vpp, ver, img, ing.GetAnnotations())
	if err != nil {
		z.Logn("[_mutate_]:", "update app version into database error,", err.Error())
		return // 更新数据库发生异常
	}
	return
}

func (aa *Serve) mutateFrontPath(ing *netv1.Ingress, svc string) (*PatchOperation, [2]string, error) {
	svc = strings.TrimSpace(svc)
	if svc == "" {
		return nil, [2]string{}, errors.New("frontend service is empty")
	}
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
	servicePort := "http" // 默认端口
	servicePath := ""
	if svc[0] == '/' {
		serviceName = C.Front3.DefaultURL // 使用默认的前端服务地址
		servicePath = svc                 // 直接使用 svc 作为路径
		if idx := strings.IndexByte(serviceName, ':'); idx >= 0 {
			servicePort = serviceName[idx+1:]
			serviceName = serviceName[:idx]
		}
	} else {
		if idx := strings.IndexByte(serviceName, ':'); idx >= 0 {
			servicePort = serviceName[idx+1:]
			serviceName = serviceName[:idx]
		}
		if idx := strings.IndexByte(servicePort, '/'); idx >= 0 {
			servicePath = servicePort[idx:]
			servicePort = servicePort[:idx]
		}
	}
	// 检查和注入 rules 信息
	if rule.HTTP == nil {
		rule.HTTP = &netv1.HTTPIngressRuleValue{}
	}
	if rule.HTTP.Paths == nil {
		rule.HTTP.Paths = []netv1.HTTPIngressPath{}
	}
	for _, path := range rule.HTTP.Paths {
		if servicePath != "" && path.Path == servicePath || path.Backend.Service != nil && path.Backend.Service.Name == serviceName {
			if servicePath != "" {
				return nil, [2]string{rule.Host, servicePath}, nil // 前端服务已经存在
			} else {
				return nil, [2]string{rule.Host, path.Path}, nil // 前端服务已经存在
			}
		}
	}
	if servicePath == "" {
		servicePath = "/" // 默认路径
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
	if servicePort == "http" { // 大部分特殊情况，优先处理
		path.Backend.Service.Port.Name = servicePort // 端点名称
	} else if port, err := strconv.Atoi(servicePort); err == nil {
		path.Backend.Service.Port.Number = int32(port) // 端口号
	} else {
		path.Backend.Service.Port.Name = servicePort // 端点名称
	}
	rule.HTTP.Paths = append(rule.HTTP.Paths, path)
	if len(rule.HTTP.Paths) == 1 {
		// 之前没有 http 节点，新增一个 http 节点
		return &PatchOperation{
			Op:    "replace",
			Path:  "/spec/rules/0/http",
			Value: rule.HTTP,
		}, [2]string{rule.Host, path.Path}, nil
	}
	// 之前已有 http 节点，增加一个 path 节点
	return &PatchOperation{
		Op:    "add",
		Path:  "/spec/rules/0/http/paths/-",
		Value: path,
	}, [2]string{rule.Host, path.Path}, nil
}
