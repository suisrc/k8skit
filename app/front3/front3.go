package front3

// 为 前端 提供 静态文件服务， 包括 s3 CDN 加速， 前端镜像缓存， http 前端缓存
// 是 front3 服务的 核心部分

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"k8skit/app/registry"
	"k8skit/app/s3cdn"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/suisrc/zgg/app/front2"
	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
)

type AppCache struct {
	Key     string
	AppInfo FrontaDO     // 应用, 不存在共享情况
	Version FrontvDO     // 版本, 存在共享的情况
	Handler http.Handler // *front2.IndexApi
	LastMod int64        // 最后访问时间
	IsLocal bool         // 是本地化
	Abspath string       // 绝对路径
}

func (aa *Serve) CleanCaches() {
	z.Println("[_front3_]: clean caches ================", aa._CacheCC)
	aa._CacheMU.Lock()
	defer aa._CacheMU.Unlock()
	now := time.Now().Unix()
	// 删除超期缓存
	clsmap := map[string]bool{}
	extmap := map[string]bool{}
	aa.CacheApp.Range(func(key, value any) bool {
		api := value.(*AppCache)
		if now-api.LastMod > aa.Interval {
			z.Println("[_front3_]: clean cache1 ================", key, api.AppInfo.App.String, api.Version.Ver)
			aa.CacheApp.Delete(key)
			if api.Abspath != "" {
				clsmap[api.Abspath] = true
			}
		} else if api.Abspath != "" {
			extmap[api.Abspath] = true
		}
		return true
	})
	// 需要检索路径是否被占用
	for path := range clsmap {
		if has, _ := extmap[path]; has {
			continue // 路径对象的缓存，还有应用使用，不能删除
		}
		z.Println("[_front3_]: clean cache2 ================", path)
		os.RemoveAll(path)
	}
}

func (aa *Serve) CleanerStart(interval time.Duration) error {
	if aa._CacheTT != nil {
		return errors.New("cleaner is working") // 定时清理运行中
	}
	aa._CacheTT = time.NewTicker(interval)
	go func() {
		for range aa._CacheTT.C {
			aa._CacheCC += 1
			go aa.CleanCaches()
		}
	}()
	return nil
}

func (aa *Serve) CleanerClose() {
	if aa._CacheTT != nil {
		aa._CacheTT.Stop()
		aa._CacheTT = nil
	}
}

func (aa *Serve) Stop() {
	aa.CleanerClose()
}

//=============================================================================================================================
//=============================================================================================================================
//=============================================================================================================================

// Serve 索引服务
func (aa *Serve) ServeS3(rw http.ResponseWriter, rr *http.Request) {
	host := rr.Host //  请求的域名
	apps, err := aa.AppRepo.GetAllByDomain(host)
	if err != nil {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "application query error: "+host+", "+err.Error(), http.StatusInternalServerError)
		return
	} else if len(apps) == 0 {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "application not found: "+host, http.StatusNotFound)
		return
	}
	if len(apps) > 1 { // Priority 降序排序
		slices.SortFunc(apps, func(l, r FrontaDO) int { return strings.Compare(r.Priority.String, l.Priority.String) })
	}
	// 通过 rootdir 确定 path
	path := rr.URL.Path
	var app *FrontaDO
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
		http.Error(rw, "application path not found: "+host, http.StatusNotFound)
		return
	}
	if app.Disable {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "application disabled: "+host, http.StatusNotFound)
		return
	}
	// 浏览器指定了版本，则优先使用
	_ver := rr.URL.Query().Get("version") // 打开特定的版本
	if _ver == "" {
		if ref := rr.Referer(); ref == "" {
			// pass
		} else if ref, err := url.Parse(ref); err == nil {
			_ver = ref.Query().Get("version")
		}
	}
	// 数据库指定了版本, 则优先使用
	if _ver == "" && app.Ver.String != "" {
		_ver = app.Ver.String
	}
	// 如果未指定版本，使用当前系统最新版本
	ver, err := aa.VerRepo.GetTop1ByVppAndVer(app.GVP(), _ver)
	if err != nil {
		if err == sql.ErrNoRows {
			rw.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.Error(rw, "application version not found: "+host+", "+_ver, http.StatusNotFound)
			return
		}
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "application version query error: "+host+", "+err.Error(), http.StatusInternalServerError)
		return
	}
	// 记录版本信息到请求头上
	rw.Header().Set("X-Front3-Ver", ver.Vpp+"; version="+ver.Ver)
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
	key := fmt.Sprintf("%d-%d-%s", app.ID, ver.ID, ver.Ver)
	var api *AppCache
	if cac, _ := aa.CacheApp.Load(key); cac != nil {
		api = cac.(*AppCache)
	}
	cls := false
	if api != nil {
		if ver.ReCache.Bool {
			// 标记强制刷新 LOC
			z.Println("[_front3_]: LOC forc refresh, delete cache:", key)
			aa.CacheApp.Delete(key)
			cls = true
			// 需要强制刷新本地缓存，如果有多实例的情况
		} else if ver.CdnPush.Bool && ver.CdnRenew.Bool {
			// 标记强制刷新 CDN
			z.Println("[_front3_]: CDN mode rewrite, delete cache:", key)
			aa.CacheApp.Delete(key)
			cls = true
		} else if ver.CdnPush.Bool == api.IsLocal {
			// 确认 CDN 和 LOC 模式是否 发生了切换
			z.Println("[_front3_]: CDN mode changed, delete cache:", key)
			aa.CacheApp.Delete(key)
			cls = true
		} else if !api.IsLocal {
			// do nohting, pass
		} else if _, err := os.Stat(api.Abspath); err != nil {
			// 缓存文件不存在，有可能被其他应用或者人工删除，重建, 一般同步缓存的时候，会删除该内容
			z.Println("[_front3_]: LOC mode, cache no found:", key)
			aa.CacheApp.Delete(key)
			api = nil // 缓存已经没有了，不需要再次清理了
		}
	}
	if api == nil || cls {
		// 没有缓存，或者缓存已经被标记需要清理了
		aa._CacheMU.Lock()
		defer aa._CacheMU.Unlock()
		if cac, _ := aa.CacheApp.Load(key); cac != nil {
			api = cac.(*AppCache) // 缓存已经重新建立，不需要重复建立
		} else {
			if api != nil && api.Abspath != "" {
				os.RemoveAll(api.Abspath) // 清理本地缓存
			}
			// 重新建立缓存
			api = aa.InitApi(rw, rr, &AppCache{Key: key, AppInfo: *app, Version: *ver})
			if api == nil {
				return // 无法处理， 不能创建 api, InitApi 中已经返回异常内容
			}
			api.LastMod = time.Now().Unix() // 防止被清理
			aa.CacheApp.Store(key, api)     // 重新建立缓存
		}
	}
	if z.IsDebug() || C.Front3.Debug {
		z.Println("[_front3_]:", key, app.App.String, "[", ver.Vpp, "] ->", rr.URL.Path)
	}
	api.LastMod = time.Now().Unix()
	api.Handler.ServeHTTP(rw, rr)
}

//=============================================================================================================================
//=============================================================================================================================
//=============================================================================================================================

func (aa *Serve) InitApi(rw http.ResponseWriter, rr *http.Request, av *AppCache) *AppCache {
	config := front2.Config{
		TmplRoot:   av.Version.TPRoot.String,
		TmplSuffix: front2.C.Front2.TmplSuffix,
		TmplPrefix: front2.C.Front2.TmplPrefix,
	}
	{
		index := av.Version.IndexPath.String
		if index == "" {
			index = front2.C.Front2.Index // 默认值
		}
		indexs := zc.StrMap{}
		if av.Version.Indexs.String != "" {
			indexs.Set(av.Version.Indexs.String)
		}
		routers := zc.StrMap{}
		if av.AppInfo.Routers.String != "" {
			routers.Set(av.AppInfo.Routers.String)
		}
		config.Index = index
		config.Indexs = indexs
		config.Routers = routers
	}
	// 处理本地缓存目录
	abspath, err := filepath.Abs(filepath.Join(aa.RegConfig.OutPath, av.Version.Vpp, av.Version.Ver))
	if err != nil {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "application local path error: "+rr.Host+" ["+abspath+"] "+err.Error(), http.StatusInternalServerError)
		return nil // 本地缓存地址无效
	}
	// 强制重置缓存
	if av.Version.ReCache.Bool {
		os.RemoveAll(abspath) // 可以忽略错误
		av.Version.ReCache = sql.NullBool{Bool: false, Valid: true}
		aa.VerRepo.UpdateCacInfo(&av.Version)
		defer aa.NoticeSyncHook("delete.cache", z.HA{"key": av.Key}) // 结束后，需要通知所有实例清理缓存
	}
	// 确定是否为CDN模式
	if av.Version.CdnName.String != "" && av.Version.CdnPush.Bool && !av.Version.CdnRenew.Bool {
		// 直接使用 CDN 模式返回, CDN 存在，且不需要重新更新
		handler := front2.NewApi(nil, config, fmt.Sprintf("[_front3_]-%d-%d", av.AppInfo.ID, av.Version.ID))
		av.Handler = handler
		s3cdn.InitCdnServe(handler, av.Version.CdnName.String, av.Version.CdnPath.String, av.Version.Vpp, av.Version.Ver)
		return av // CDN模式， 直接返回
	}
	// 验证镜像文件地址
	if av.Version.Image.String == "" {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "application image empty: "+rr.Host, http.StatusInternalServerError)
		return nil // 没有镜像地址
	}
	// 获取前端文件镜像, 在本地部署前端资源文件 //os.WriteFile(filepath.Join(abspath, "aname"), []byte(time.Now().Format(time.RFC3339)), 0644)
	if _, err := os.Stat(abspath); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(abspath, 0644); err != nil {
			rw.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.Error(rw, "application local path error: "+rr.Host+" ["+abspath+"] "+err.Error(), http.StatusInternalServerError)
			return nil // 无法创建缓存文件夹
		}
		// 优先使用 cdn 缓存
		completed := false
		tgzobject := filepath.Join(aa.CdnConfig.RootDir, av.Version.Vpp, av.Version.Ver) + ".tgz"
		var s3cli *minio.Client = nil
		if av.Version.CdnCache.Bool && av.Version.CdnRenew.Bool {
			// 获取 s3 client， 以便用于后面更新
			s3cli, _ = s3cdn.GetClient(context.Background(), &aa.CdnConfig)
		} else if av.Version.CdnCache.Bool {
			// 优先尝试使用 cdn 缓存
			if s3cli, err = s3cdn.GetClient(context.Background(), &aa.CdnConfig); err != nil {
				z.Println("[_front3_]: used cdn cache error:", tgzobject, err.Error())
			} else if obj, err := s3cli.GetObject(context.TODO(), aa.CdnConfig.Bucket, tgzobject, minio.GetObjectOptions{}); err != nil {
				z.Println("[_front3_]: used cdn cache error:", tgzobject, err.Error())
			} else if err := registry.ExtractTgzByReader(abspath, "", obj); err != nil {
				z.Println("[_front3_]: used cdn cache error:", tgzobject, err.Error())
			} else {
				z.Println("[_front3_]: used cdn cache success:", tgzobject)
				completed = true
			}
		}
		if !completed && (strings.HasPrefix(av.Version.Image.String, "http://") || strings.HasPrefix(av.Version.Image.String, "https://")) {
			// 使用 http 获取镜像文件
			var rerr error
			if resp, err := http.Get(av.Version.Image.String); err != nil {
				z.Println("[_front3_]: download by http error:", av.Version.Vpp, av.Version.Ver, av.Version.Image.String, err.Error())
				rerr = err
			} else if err := registry.ExtractTgzByReader(abspath, av.Version.ImagePath.String, resp.Body); err != nil {
				z.Println("[_front3_]: download by http error:", av.Version.Vpp, av.Version.Ver, av.Version.Image.String, err.Error())
				rerr = err
				_ = resp.Body.Close()
			} else {
				z.Println("[_front3_]: download by http success:", av.Version.Vpp, av.Version.Ver, av.Version.Image.String)
				_ = resp.Body.Close()
				completed = true
			}
			if rerr != nil {
				rw.Header().Set("Content-Type", "text/html; charset=utf-8")
				http.Error(rw, "application download package error: "+rr.Host+", "+rerr.Error(), http.StatusInternalServerError)
				os.RemoveAll(abspath) // 删除本地缓存文件夹
				return nil            // 无法下载镜像文件
			}
		}
		if !completed {
			// 使用 registry 获取镜像文件
			cfg := registry.Config{
				Username: aa.RegConfig.Username,
				Password: aa.RegConfig.Password,
				DcrAuths: aa.RegConfig.DcrAuths,
				Image:    av.Version.Image.String,
				SrcPath:  av.Version.ImagePath.String,
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
			if err := registry.ExportFile(&cfg); err != nil {
				z.Println("[_front3_]: export image file error:", av.Version.Vpp, av.Version.Ver, av.Version.Image.String, err.Error())
				rw.Header().Set("Content-Type", "text/html; charset=utf-8")
				http.Error(rw, "application pull image error: "+rr.Host+", "+err.Error(), http.StatusInternalServerError)
				os.RemoveAll(abspath) // 删除本地缓存文件夹
				return nil            // 无法提出镜像文件
			}
			if s3cli != nil {
				// S3终端已经被打开, 上传缓存文件
				pr, pw := io.Pipe()
				go func() {
					if err := registry.CreateTgzByWriter(abspath, pw); err != nil {
						_ = pw.CloseWithError(err)
						return
					}
					_ = pw.Close()
				}()
				if _, err := s3cli.PutObject(context.TODO(), aa.CdnConfig.Bucket, tgzobject, pr, -1, minio.PutObjectOptions{}); err != nil {
					z.Println("[_front3_]: error, upload cdn cache error:", tgzobject, err.Error())
					_ = pr.CloseWithError(err)
				} else {
					z.Println("[_front3_]: upload cdn cache success:", tgzobject)
					_ = pr.Close()
				}
				// s3cli.Close()
			}
		}
	} else if err != nil {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "application local path error: "+rr.Host+" ["+abspath+"] "+err.Error(), http.StatusInternalServerError)
		return nil // 查询本地缓存发生异常
	} else {
		z.Println("[_front3_]: local path, exist:", abspath)
	}
	av.Abspath = abspath
	handler := front2.NewApi(os.DirFS(abspath), config, fmt.Sprintf("[_front3_]-%d-%d", av.AppInfo.ID, av.Version.ID))
	av.Handler = handler
	// 使用 CDN 内容返回
	if av.Version.CdnPush.Bool {
		// 上传到 cdn， 部署CDN
		cfg := aa.CdnConfig // 赋值了新对象
		cfg.Rewrite = av.Version.CdnRenew.Bool
		err = s3cdn.UploadToS3(handler.HttpFS, handler.FileFS, &handler.Config, &cfg, av.Version.Vpp, av.Version.Ver)
		if err != nil {
			z.Println("[_f3serve]: error, upload cdn err:", err.Error())
			rw.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.Error(rw, "application upload cdn error: "+rr.Host+err.Error(), http.StatusInternalServerError)
			return nil
		}
		s3cdn.InitCdnServe(handler, aa.CdnConfig.Domain, aa.CdnConfig.RootDir, av.Version.Vpp, av.Version.Ver)
		// 更新CDN信息
		av.Version.CdnName = sql.NullString{String: aa.CdnConfig.Domain, Valid: true}
		av.Version.CdnPath = sql.NullString{String: aa.CdnConfig.RootDir, Valid: true}
		av.Version.CdnRenew = sql.NullBool{Bool: false, Valid: true}
		aa.VerRepo.UpdateCdnInfo(&av.Version)
	} else {
		av.IsLocal = true // 本地模式
	}
	return av
}
