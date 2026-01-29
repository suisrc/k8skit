package front3

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
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
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
	Interval  int64                // 单位秒, 巡检间隔
	AppCache  map[string]*AppCache // sync.Map vs map[string]*AppCache & _CacheMu
	_CacheMu  sync.Mutex           // 缓存操作锁
	_ClrTime  *time.Ticker         // 定时清理缓存
	_ClrCout  int64
}

type AppCache struct {
	AppInfo AppInfoDO    // 应用, 不存在共享情况
	Version VersionDO    // 版本, 存在共享的情况
	Handler http.Handler // *front2.IndexApi
	LastMod int64        // 最后访问时间
	IsLocal bool         // 是本地化
	Abspath string       // 绝对路径
}

func (aa *F3Serve) CleanCaches() {
	z.Println("[_front3_]: clean caches ================", aa._ClrCout)
	aa._CacheMu.Lock()
	defer aa._CacheMu.Unlock()
	now := time.Now().Unix()
	// 删除超期缓存
	clsmap := map[string]bool{}
	for k, v := range aa.AppCache {
		if now-v.LastMod > aa.Interval {
			z.Println("[_front3_]: clean cache1 ================", k, v.AppInfo.App.String, v.Version.Ver)
			delete(aa.AppCache, k)
			if v.Abspath != "" {
				clsmap[v.Abspath] = true
			}
		}
	}
	// 需要检索路径是否被占用
	for path := range clsmap {
		has := false
		for _, v := range aa.AppCache {
			if v.Abspath == path {
				has = true
				break
			}
		}
		if has {
			continue
		}
		z.Println("[_front3_]: clean cache2 ================", path)
		os.RemoveAll(path)
	}
}

func (aa *F3Serve) CleanerWork(interval time.Duration) error {
	if aa._ClrTime != nil {
		return errors.New("cleaner is working") // 定时清理运行中
	}
	aa._ClrTime = time.NewTicker(interval)
	go func() {
		for range aa._ClrTime.C {
			aa._ClrCout += 1
			go aa.CleanCaches()
		}
	}()
	return nil
}

func (aa *F3Serve) CleanerStop() {
	if aa._ClrTime != nil {
		aa._ClrTime.Stop()
		aa._ClrTime = nil
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
		http.Error(rw, "application query error: "+host+", "+err.Error(), http.StatusInternalServerError)
		return
	} else if len(apps) == 0 {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "application not found: "+host, http.StatusNotFound)
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
	api, _ := aa.AppCache[key]
	if api != nil {
		// 确认 CDN 和 LOC 模式是否 发生了切换
		if ver.CdnPush.Bool && ver.CdnRenew.Bool {
			delete(aa.AppCache, key) // 删除缓存
			if api.Abspath != "" {
				os.RemoveAll(api.Abspath) // 清理缓存
			}
			api = nil
			z.Println("[_front3_]: CDN mode rewrite, delete cache:", key)
		} else if ver.CdnPush.Bool == api.IsLocal {
			delete(aa.AppCache, key) // 删除缓存
			if api.Abspath != "" {
				os.RemoveAll(api.Abspath) // 清理缓存
			}
			api = nil
			z.Println("[_front3_]: CDN mode changed, delete cache:", key)
		} else if api.IsLocal {
			if _, err := os.Stat(api.Abspath); err != nil {
				// 缓存文件不存在，有可能被其他应用或者人工删除，重建
				api = nil
				z.Println("[_front3_]: LOC mode, cache no found:", key)
			}
		}
	}

	if api == nil {
		// 没有缓存，则创建一个
		aa._CacheMu.Lock()
		defer aa._CacheMu.Unlock()
		if api, _ = aa.AppCache[key]; api != nil {
			// api 已经存在，跳过
		} else if api = aa.InitApi(rw, rr, &AppCache{AppInfo: *app, Version: *ver}); api != nil {
			api.LastMod = time.Now().Unix() // 防止被清理
			aa.AppCache[key] = api
		} else {
			return // 无法处理， 不能创建 api
		}
	}
	if z.IsDebug() || C.Front3.Debug {
		z.Println("[_front3_]:", key, app.App.String, "[", ver.Vpp, "] ->", rr.URL.Path)
	}
	api.LastMod = time.Now().Unix()
	api.Handler.ServeHTTP(rw, rr)
}

func (aa *F3Serve) InitApi(rw http.ResponseWriter, rr *http.Request, av *AppCache) *AppCache {
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
	if av.Version.CdnName.String != "" && av.Version.CdnPush.Bool && !av.Version.CdnRenew.Bool {
		// 直接使用 CDN 模式返回, CDN 存在，且不需要重新更新
		handler := front2.NewApi(nil, config, fmt.Sprintf("[_front3_]-%d-%d", av.AppInfo.ID, av.Version.ID))
		av.Handler = handler
		s3cdn.InitCdnServe(handler, av.Version.CdnName.String, av.Version.CdnPath.String, av.Version.Vpp, av.Version.Ver)
		return av
	}
	// 验证镜像文件地址
	if av.Version.Image.String == "" {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "application image empty: "+rr.Host, http.StatusInternalServerError)
		return nil // 没有镜像地址
	}
	// 处理本地缓存目录
	outpath := filepath.Join(aa.RegConfig.OutPath, av.Version.Vpp, av.Version.Ver)
	abspath, err := filepath.Abs(outpath)
	if err != nil {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "application local path error: "+rr.Host+" ["+outpath+"] "+err.Error(), http.StatusInternalServerError)
		return nil // 本地缓存地址无效
	}
	if av.Version.ReCache.Bool {
		// 强制重新缓存
		os.RemoveAll(abspath)
		av.Version.ReCache = sql.NullBool{Bool: false, Valid: true}
		aa.VerRepo.UpdateCacInfo(&av.Version)
	}
	// 获取前端文件镜像, 在本地部署前端资源文件 //os.WriteFile(filepath.Join(abspath, "aname"), []byte(time.Now().Format(time.RFC3339)), 0644)
	if _, err := os.Stat(abspath); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(abspath, 0644); err != nil {
			rw.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.Error(rw, "application local path error: "+rr.Host+" ["+outpath+"] "+err.Error(), http.StatusInternalServerError)
			return nil // 无法创建缓存文件夹
		}
		// 优先使用 cdn 缓存
		abort := false
		tgzobj := filepath.Join(aa.CdnConfig.RootDir, av.Version.Vpp, av.Version.Ver) + ".tgz"
		var s3cli *minio.Client = nil
		if av.Version.CdnCache.Bool && av.Version.CdnRenew.Bool {
			// 获取 s3 client， 以便用于后面更新
			s3cli, _ = s3cdn.GetClient(context.Background(), &aa.CdnConfig)
		} else if av.Version.CdnCache.Bool {
			// 优先尝试使用 cdn 缓存
			if s3cli, err = s3cdn.GetClient(context.Background(), &aa.CdnConfig); err != nil {
				z.Println("[_front3_]: used cdn cache error:", tgzobj, err.Error())
			} else if obj, err := s3cli.GetObject(context.TODO(), aa.CdnConfig.Bucket, tgzobj, minio.GetObjectOptions{}); err != nil {
				z.Println("[_front3_]: used cdn cache error:", tgzobj, err.Error())
			} else if err := registry.ExtractTgzByReader(abspath, obj); err != nil {
				z.Println("[_front3_]: used cdn cache error:", tgzobj, err.Error())
			} else {
				z.Println("[_front3_]: used cdn cache success:", tgzobj)
				abort = true
			}
		}
		if !abort {
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
				if _, err := s3cli.PutObject(context.TODO(), aa.CdnConfig.Bucket, tgzobj, pr, -1, minio.PutObjectOptions{}); err != nil {
					z.Println("[_front3_]: error, upload cdn cache error:", tgzobj, err.Error())
					_ = pr.CloseWithError(err)
				} else {
					z.Println("[_front3_]: upload cdn cache success:", tgzobj)
					_ = pr.Close()
				}
			}
		}

	} else if err != nil {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(rw, "application local path error: "+rr.Host+" ["+outpath+"] "+err.Error(), http.StatusInternalServerError)
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
