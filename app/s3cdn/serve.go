package s3cdn

import (
	"io"
	"maps"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/suisrc/zgg/app/front2"
	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
)

// 提供 S3 索引服务
func InitCdnServe(api *front2.IndexApi, domain, rootdir string, appname, version string) {
	api.ServeFS = &ServeS3{
		LogKey:    api.LogKey,
		Index:     api.Config.Index,
		Indexs:    api.Config.Indexs,
		IndexsKey: api.IndexsKey,
		Domain:    domain,
		RootDir:   rootdir,
		AppName:   appname,
		Version:   version,
	}
}

type ServeS3 struct {
	LogKey    string
	Index     string
	Indexs    map[string]string
	IndexsKey []string
	Domain    string
	RootDir   string
	AppName   string
	Version   string
}

func (aa *ServeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := ""
	rpath := ""
	if ext := filepath.Ext(r.URL.Path); ext != "" {
		path = r.URL.Path // 文件资源
	} else {
		rpath = r.Header.Get("X-Req-RootPath")
		if rpath != "" {
			// 寻找指定索引文件
			path, _ = aa.Indexs[rpath]
		} else {
			// 通过匹配查询索引文件
			for _, kk := range aa.IndexsKey {
				if r.URL.Path == kk || zc.HasPrefixFold(r.URL.Path, kk+"/") {
					path = aa.Indexs[kk] // 匹配到, 使用 v 代替 index
					break
				}
			}
		}
	}
	if path == "" {
		path = aa.Index
	}
	path = aa.Domain + "/" + filepath.Join(aa.RootDir, aa.AppName, aa.Version, path)
	resp, err := http.Get(path)
	if err != nil {
		z.Println(aa.LogKey+": error, redirect to:", path, r.URL.Path, err.Error())
		http.Redirect(w, r, path, http.StatusMovedPermanently)
	} else {
		// z.Println(aa.LogKey+": redirect to:", path, r.URL.Path)
		if ctype := resp.Header.Get("Content-Type"); strings.HasPrefix(ctype, "application/octet-stream") {
			resp.Header.Set("Content-Type", "text/html; charset=utf-8")
		}
		if rpath != "" {
			w.Header().Set("X-Request-Rp", rpath) // 通过 CDN 加载的索引文件，存在 /rootpath 未替换的问题
			// X-Request-Rp 与 X-Req-RootPath 区分, 防止被意外替换, X-Req-RootPath 来自上级路由业务的内容
		}
		maps.Copy(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}
