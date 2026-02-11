package front3

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
)

// 访问权限确认
func (aa *Serve) CheckPermission(qry url.Values, role string) (string, error) {
	akk := qry.Get("ak")
	if akk == "" {
		return "", errors.New("token is empty")
	}
	akey, err := aa.AuzRepo.GetByAppKey(akk)
	if err != nil || akey == nil {
		return "", errors.New("token is invalid")
	} else if akey.Disable {
		return "", errors.New("token is disable")
	} else if perm := akey.Permiss.String; perm == "" {
		return "", errors.New("permission is empty")
	} else if role == "" {
		// pass
	} else if strings.HasSuffix(perm, ".*") && strings.HasPrefix(role, perm[0:len(perm)-1]) {
		// pass
	} else if perm == role {
		// pass
	} else {
		return "", errors.New("permission is forbidden")
	}
	return akey.Permiss.String, nil
}

// 更新应用版本
func (aa *Serve) UpdateImageVersion(zrc *z.Ctx) {
	qry := zrc.Request.URL.Query()
	if _, err := aa.CheckPermission(qry, "front3.update.image"); err != nil {
		zrc.TEXT(err.Error(), http.StatusOK)
		return
	}
	image := qry.Get("image")
	// 验证镜像是否已经存在， 防止重复更新
	if vers, err := aa.VerRepo.GetByImage(image); err == nil && len(vers) > 0 {
		zrc.TEXT("image is exist", http.StatusOK)
		return
	}
	kver := strings.SplitN(image, ":", 2)
	if len(kver) != 2 {
		zrc.TEXT("image is invalid", http.StatusOK)
		return
	}
	ikey, iver := kver[0], kver[1]
	if app := qry.Get("app"); app != "" {
		// 通过应用编码查找应用
		appInfo, err := aa.AppRepo.GetByApp(app)
		if err != nil {
			zrc.TEXT("application not found: "+app, http.StatusOK)
			return
		}
		vpp := appInfo.GVP()
		verInfo, err := aa.VerRepo.GetTop1ByVppAndVer(vpp, "") // 最新版
		if err != nil {
			zrc.TEXT("application version not found: "+app, http.StatusOK)
			return
		}
		verInfo.ID = 0
		verInfo.CdnName.Valid = false
		verInfo.CdnPath.Valid = false
		verInfo.CdnPush.Valid = false
		verInfo.CdnRenew.Valid = false
		verInfo.Image = sql.NullString{String: image, Valid: true}
		verInfo.Vpp = vpp
		verInfo.Ver = iver
		verInfo.Started = sql.NullTime{Time: time.Now(), Valid: true}
		aa.VerRepo.Insert(verInfo)
	} else {
		// 使用镜像 ikey 查询和替换应用版本， 需要考虑存在多个的情况
		vers, err := aa.VerRepo.GetByImageName(ikey)
		if err != nil || len(vers) == 0 {
			zrc.TEXT("application version not found: "+ikey, http.StatusOK)
			return
		}
		// 遍历更新应用版本
		for _, ver := range vers {
			ver.ID = 0
			ver.CdnName.Valid = false
			ver.CdnPath.Valid = false
			ver.CdnPush.Valid = false
			ver.CdnRenew.Valid = false
			ver.Image = sql.NullString{String: image, Valid: true}
			ver.Ver = iver // 只更新版本和镜像
			ver.Started = sql.NullTime{Time: time.Now(), Valid: true}
			aa.VerRepo.Insert(&ver)
		}
	}

	zrc.TEXT("ok", http.StatusOK)
}

// 删除本地 app 缓存， 接受处理, method=delete.local.cache
func (aa *Serve) DeleteLocalCache(zrc *z.Ctx) {
	qry := zrc.Request.URL.Query()
	if _, err := aa.CheckPermission(qry, "front3.delete.cache"); err != nil {
		zrc.TEXT(err.Error(), http.StatusOK)
		return
	}
	key := qry.Get("key")
	if key == "" {
		zrc.TEXT("key is empty", http.StatusOK)
		return
	}
	// [fronta.id]-[frontv.id]-[version] =fmt.Sprintf("%d-%d-%s", app.ID, ver.ID, ver.Ver)
	api, _ := aa.CacheApp.LoadAndDelete(key)
	if api == nil {
		zrc.TEXT("app cache not found", http.StatusOK)
		return
	}
	if abspath := api.(*AppCache).Abspath; abspath != "" {
		os.RemoveAll(abspath)
	}
	zrc.TEXT("ok", http.StatusOK)
}

// ==============================================================================================

func (aa *Serve) AnswerSyncHook(source, method string, zrc *z.Ctx) {
	data := map[string]any{}
	if C.Front3.SyncToken == "" {
		z.Println("[_syncfg_]:", "answer sync token is empty") // 未配置同步秘钥
		zrc.TEXT("answer sync token is empty", http.StatusOK)
		return
	} else if _, err := z.ReadBody(zrc.Request, &data); err != nil {
		z.Println("[_syncfg_]:", "read body error: "+err.Error()) // 解析 body 异常
		zrc.TEXT("read body error", http.StatusOK)
		return
	} else if token, _ := data["token"]; token != C.Front3.SyncToken {
		z.Println("[_syncfg_]:", "answer sync token is not equal") // 同步秘钥不一致
		zrc.TEXT("answer sync token is not equal", http.StatusOK)
		return
	}
	z.Println("[_syncfg_]:", "answer sync config,", source, "-> ", method)
	switch method {
	case "delete.cache":
		key, _ := data["key"].(string)
		if key == "" {
			z.Println("[_syncfg_]: answer sync config, key is empty,", key)
			zrc.TEXT("ok", http.StatusOK)
			return
		}
		api, _ := aa.CacheApp.LoadAndDelete(key)
		if api == nil {
			z.Println("[_syncfg_]: answer sync config, app cache not found,", key)
			zrc.TEXT("ok", http.StatusOK)
			return
		}
		if abspath := api.(*AppCache).Abspath; abspath != "" {
			os.RemoveAll(abspath)
			z.Println("[_syncfg_]: answer sync config, clear path, ", abspath)
		}
	default:
		z.Println("[_syncfg_]: answer sync config, method not found,", method)
	}
	zrc.TEXT("ok", http.StatusOK)
}

// ==============================================================================================

// 删除本地 app 缓存， 通知处理， 这个同步只针对 StatefulSet 场景优化
func (aa *Serve) NoticeSyncHook(key string, data map[string]any) {
	go aa.NoticeASyncHook(key, data) // 异步通知
}

func (aa *Serve) NoticeASyncHook(key string, data map[string]any) {
	if C.Front3.SyncToken == "" {
		z.Println("[_syncfg_]: sync token is empty")
		return
	} else if C.Front3.SyncServe == "" {
		z.Println("[_syncfg_]: sync serve is empty")
		return
	} else if C.Front3.WebHookPath == "" {
		z.Println("[_syncfg_]: sync path(WebHookPath) is empty")
		return
	}
	data["token"] = C.Front3.SyncToken
	// 同步端点信息
	syncpath := C.Front3.WebHookPath
	if syncpath[0] != '/' {
		syncpath = "/" + syncpath // 补充路径前缀
	}
	syncport := strconv.Itoa(z.C.Server.Port) // 服务节点的IP端口
	selfhost := zc.GetLocAreaIp()             // 当前节点局域网IP，用于排除使用
	// 同步端点列表
	syncnode := []string{}
	if strings.HasPrefix(C.Front3.SyncServe, "http://") || strings.HasPrefix(C.Front3.SyncServe, "https://") {
		// 直接指定了同步地址
		syncnode = strings.Fields(C.Front3.SyncServe)
	} else if ips, err := net.LookupIP(C.Front3.SyncServe); err != nil {
		z.Println("[_syncfg_]: sync serve lookup [", C.Front3.SyncServe, "] error:", err.Error())
		return
	} else {
		// 通过DNS服务，查询同步地址，注意必须使用 headless 服务
		for _, ip := range ips {
			if synchost := ip.String(); synchost != selfhost {
				syncnode = append(syncnode, "http://"+synchost+":"+syncport+syncpath)
			}
		}
	}
	z.Println("[_syncfg_]: notice sync config to ", syncnode)
	// 进行端点同步
	for _, uripath := range syncnode {
		if idx := strings.IndexByte(uripath, '?'); idx > 0 {
			uripath += "&method=" + key
		} else {
			uripath += "?method=" + key
		}
		uripath += "&source=" + selfhost // 同步来源

		if bts, err := json.Marshal(data); err != nil {
			z.Printf("[_syncfg_]: json.Marshal error, %s, %v", err.Error(), data)
		} else if resp, err := http.Post(uripath, "application/json", bytes.NewBuffer(bts)); err != nil {
			z.Printf("[_syncfg_]: http.Post error, %s, %s", err.Error(), uripath)
		} else {
			bts, _ = io.ReadAll(resp.Body)
			resp.Body.Close()
			z.Printf("[_syncfg_]: sync success, %s, %d, %s", uripath, resp.StatusCode, string(bts))
		}
	}
}

// var (
// 	serve_name = ""
// )

// func GetServeName() string {
// 	if serve_name != "" {
// 		return serve_name
// 	}
// 	// 通过 /etc/hosts 获取局域网地址
// 	if bts, err := os.ReadFile("/etc/hosts"); err != nil {
// 		zc.Printf1("unable to read /etc/hosts: %s", err.Error())
// 	} else {
// 		for line := range strings.SplitSeq(string(bts), "\n") {
// 			if strings.HasPrefix(line, "#") {
// 				continue
// 			}
// 			ips := strings.Fields(line)
// 			if len(ips) < 2 || ips[0] == "127.0.0.1" {
// 				continue
// 			}
// 			for _, name := range ips[1:] {
// 				if strings.HasSuffix(name, ".svc.cluster.local") {
// 					serve_name = name
// 					break // 找到与 hostname 匹配的 IP
// 				}
// 			}
// 			if serve_name != "" {
// 				break
// 			}
// 		}
// 	}
// 	if serve_name == "" {
// 		serve_name = zc.GetHostname() // 无法解析
// 	}
// 	return serve_name
// }
