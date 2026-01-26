package front3

import (
	"database/sql"
	"errors"
	"fmt"
	"k8skit/app/registry"
	"k8skit/app/s3cdn"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/suisrc/zgg/app/front2"
	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
)

type F3Serve struct {
	CdnConfig s3cdn.Config
	RegConfig registry.Config
	AppRepo   *AppInfoRepo
	VerRepo   *VersionRepo
	KeyRepo   *AccessKeyRepo
	CacheApi  map[string]*ApiData
	Interval  int64        // 单位秒
	_CacheMu  sync.Mutex   // 缓存操作锁
	_ClsTime  *time.Ticker // 定时清理缓存
}

type ApiData struct {
	AppInfo AppInfoDO // 应用
	Version VersionDO // 版本
	Server  *front2.IndexApi
	LastMod int64  // 最后访问时间
	IsLocal bool   // 是本地化
	Abspath string // 绝对路径
}

func (aa *F3Serve) CleanCaches() {
	z.Println("[_front3_]: clean caches ================================")
	aa._CacheMu.Lock()
	defer aa._CacheMu.Unlock()
	now := time.Now().Unix()
	for k, v := range aa.CacheApi {
		if now-v.LastMod > aa.Interval {
			z.Println("[_front3_]: clean cache1 ================", k, v.AppInfo.App.String, v.Abspath)
			delete(aa.CacheApi, k)
			if v.Abspath != "" {
				os.RemoveAll(v.Abspath) // 清理缓存
			}
		}
	}
}

func (aa *F3Serve) CleanerWork(interval time.Duration) error {
	if aa._ClsTime != nil {
		return errors.New("cleaner is working") // 定时清理运行中
	}
	z.Println("[_front3_]: cache cleaner ===============================")
	aa._ClsTime = time.NewTicker(interval)
	go func() {
		for range aa._ClsTime.C {
			go aa.CleanCaches()
		}
	}()
	return nil
}

func (aa *F3Serve) CleanerStop() {
	if aa._ClsTime != nil {
		aa._ClsTime.Stop()
		aa._ClsTime = nil
	}
}

func (aa *F3Serve) Stop() {
	if aa.AppRepo.Database != nil {
		aa.AppRepo.Database.Close()
	}
	aa.CleanerStop()
}

// F3Serve 索引服务
func (aa *F3Serve) ServeHTTP(rw http.ResponseWriter, rr *http.Request) {
	host := rr.Host //  请求的域名
	apps, err := aa.AppRepo.GetAllByDomain(host)
	if err != nil {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "Application query error: "+host+", "+err.Error(), http.StatusInternalServerError)
		return
	} else if len(apps) == 0 {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "Application not found: "+host, http.StatusNotFound)
		return
	}
	if len(apps) > 1 { // Priority 降序排序
		slices.SortFunc(apps, func(l, r AppInfoDO) int { return strings.Compare(r.Priority.String, l.Priority.String) })
	}
	// 通过 rootdir 确定 path
	path := rr.URL.Path
	var app *AppInfoDO
	for _, vvv := range apps {
		rootdir := vvv.RootDir.String
		if rootdir == "" || rootdir == "/" {
			app = &vvv
			break
		}
		if rootdir[len(rootdir)-1] == '/' {
			rootdir = rootdir[:len(rootdir)-1]
		}
		if rootdir == path || strings.HasPrefix(path, rootdir+"/") {
			app = &vvv
			break
		}
	}
	if app == nil {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "Application path not found: "+host, http.StatusNotFound)
		return
	}
	if app.Disable {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "Application disabled: "+host, http.StatusNotFound)
		return
	}
	// 确认应用的最新版本
	_ver := rr.URL.Query().Get("version") // 打开特定的版本
	if _ver == "" {
		if ref := rr.Referer(); ref == "" {
			// pass
		} else if ref, err := url.Parse(ref); err == nil {
			_ver = ref.Query().Get("version")
		}
	}
	ver, err := aa.VerRepo.GetTop1ByAidAndVer(app.ID, _ver)
	if err != nil {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "Application version query error: "+host+", "+err.Error(), http.StatusInternalServerError)
		return
	}
	// 优先返回已经存在的内容
	if ver.IndexHtml.String != "" {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		if ver.Started.Valid {
			http.ServeContent(rw, rr, "index.html", ver.Started.Time, strings.NewReader(ver.IndexHtml.String))
		} else {
			http.ServeContent(rw, rr, "index.html", time.Now(), strings.NewReader(ver.IndexHtml.String))
		}
		return
	}
	key := fmt.Sprintf("%d-%s", ver.ID, ver.Ver.String)
	api, ok := aa.CacheApi[key]
	if ok && api != nil {
		// 确认 CDN 和 LOC 模式是否 发生了切换
		if ver.CdnUse.Bool && ver.CdnRew.Bool {
			delete(aa.CacheApi, key) // 删除缓存
			api, ok = nil, false
			z.Println("[_front3_]: CDN mode rewrite, delete cache:", key)
		} else if ver.CdnUse.Bool == api.IsLocal {
			delete(aa.CacheApi, key) // 删除缓存
			api, ok = nil, false
			z.Println("[_front3_]: CDN mode changed, delete cache:", key)
		}
	}

	if !ok || api == nil {
		// 没有缓存，则创建一个
		aa._CacheMu.Lock()
		defer aa._CacheMu.Unlock()
		if api, ok = aa.CacheApi[key]; ok && api != nil {
			// pass 已经存在，跳过
		} else if api = aa.NewApi(rw, rr, *app, *ver); api != nil {
			api.LastMod = time.Now().Unix() // 防止被清理
			aa.CacheApi[key] = api
		}
	}
	if z.IsDebug() || C.Front3.Debug {
		z.Println("[_front3_]:", key, api.AppInfo.App.String, "->", rr.URL.Path)
	}
	api.LastMod = time.Now().Unix()
	api.Server.ServeHTTP(rw, rr)
}

func (aa *F3Serve) NewApi(rw http.ResponseWriter, rr *http.Request, app AppInfoDO, ver VersionDO) *ApiData {
	api := &ApiData{AppInfo: app, Version: ver}

	index := ver.IndexPath.String
	if index == "" {
		index = front2.C.Front2.Index // 默认值
	}
	indexs := zc.StrMap{}
	if ver.Indexs.String != "" {
		indexs.Set(ver.Indexs.String)
	}
	routers := zc.StrMap{}
	if app.Routers.String != "" {
		routers.Set(app.Routers.String)
	}
	config := front2.Config{
		Index:      ver.IndexPath.String,
		Indexs:     indexs,
		Routers:    routers,
		TmplRoot:   ver.TPRoot.String,
		TmplSuffix: front2.C.Front2.TmplSuffix,
		TmplPrefix: front2.C.Front2.TmplPrefix,
	}
	if ver.CdnName.String != "" && ver.CdnUse.Bool && !ver.CdnRew.Bool {
		// 直接使用 CDN 模式返回
		cdn := s3cdn.NewApi(index, indexs, ver.CdnName.String, ver.CdnPath.String, //
			fmt.Sprintf("[_s3serve]-%d", ver.ID), app.App.String, ver.Ver.String)
		api.Server = &front2.IndexApi{
			Config:    config,
			IndexsKey: cdn.IndexsKey,
			ServeFS:   cdn,
			// RouterKey: []string{}, // 基本是禁用路由功能
		}
		return api
	}
	// 验证镜像文件地址
	if ver.Image.String == "" {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "Application image empty: "+rr.Host, http.StatusInternalServerError)
		return nil
	}
	// 处理本地缓存目录
	outpath := filepath.Join(aa.RegConfig.OutPath, app.App.String, ver.Ver.String)
	abspath, err := filepath.Abs(outpath)
	if err != nil {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "Application local path error: "+rr.Host+" ["+outpath+"] "+err.Error(), http.StatusInternalServerError)
		return nil
	}
	// 获取前端文件镜像, 在本地部署前端资源文件 //os.WriteFile(filepath.Join(abspath, "aname"), []byte(time.Now().Format(time.RFC3339)), 0644)
	if _, err := os.Stat(abspath); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(abspath, 0644); err != nil {
			rw.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.Error(rw, "Application local path error: "+rr.Host+" ["+outpath+"] "+err.Error(), http.StatusInternalServerError)
			return nil
		}
		cfg := registry.Config{
			Username: aa.RegConfig.Username,
			Password: aa.RegConfig.Password,
			DcrAuths: aa.RegConfig.DcrAuths,
			Image:    ver.Image.String,
			SrcPath:  ver.ImagePath.String,
			OutPath:  abspath,
		}
		// 替换镜像地址
		if len(C.Front3.ImageMaps) > 0 {
			for kk, vv := range C.Front3.ImageMaps {
				if strings.HasPrefix(cfg.Image, kk) {
					cfg.Image = vv + cfg.Image[len(kk):]
					break // 匹配到，更换镜像地址
				}
			}
		}
		// 提取镜像文件
		if err := registry.ExtractImageFile(&cfg); err != nil {
			rw.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.Error(rw, "Application pull image error: "+rr.Host+", "+err.Error(), http.StatusInternalServerError)
			os.RemoveAll(abspath) // 删除本地缓存文件夹
			return nil
		}
	} else if err != nil {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "Application local path error: "+rr.Host+" ["+outpath+"] "+err.Error(), http.StatusInternalServerError)
		return nil
	} else {
		z.Println("[_front3_]: local path, exist:", abspath)
	}
	api.Abspath = abspath
	api.Server = front2.NewApi(os.DirFS(abspath), config, fmt.Sprintf("[_front2_]-%d", ver.ID))
	// 使用 CDN 内容返回
	if ver.CdnUse.Bool {
		// 上传到 cdn， 部署CDN
		cfg := aa.CdnConfig // 赋值了新对象
		cfg.Rewrite = ver.CdnRew.Bool
		err = s3cdn.UploadToS3(api.Server.HttpFS, api.Server.FileFS, &api.Server.Config, &cfg, app.App.String, ver.Ver.String)
		if err != nil {
			z.Println("[_f3serve]: error, upload cdn err:", err.Error())
			rw.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.Error(rw, "Application upload cdn error: "+rr.Host+err.Error(), http.StatusInternalServerError)
			return nil
		}
		api.Server.ServeFS = &s3cdn.S3IndexApi{
			Index:     api.Server.Config.Index,
			Indexs:    api.Server.Config.Indexs,
			IndexsKey: api.Server.IndexsKey,
			Domain:    aa.CdnConfig.Domain,
			RootDir:   aa.CdnConfig.RootDir,
			LogKey:    fmt.Sprintf("[_s3serve]-%d", ver.ID),
			AppName:   app.App.String,
			Version:   ver.Ver.String,
		}
		// 更新CDN信息
		api.Version.CdnName = sql.NullString{String: aa.CdnConfig.Domain, Valid: true}
		api.Version.CdnPath = sql.NullString{String: aa.CdnConfig.RootDir, Valid: true}
		api.Version.CdnRew = sql.NullBool{Bool: false, Valid: true}
		aa.VerRepo.UpdateCdnInfo(&api.Version)
	} else {
		api.IsLocal = true // 本地模式
	}
	return api
}
