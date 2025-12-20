package assets

import (
	"kube-sidecar/app"
	"net/http"
	"os"

	"github.com/suisrc/zgg/z"
)

func init() {
	z.Register("10-www", func(srv z.IServer) z.Closed {
		www := http.FileServerFS(os.DirFS(app.C.DirWWW))
		api := z.Inject(srv.GetSvcKit(), &AssetsApi{www})
		srv.AddRouter("", api.index)
		return nil
	})
}

type AssetsApi struct {
	www http.Handler
}

func (aa *AssetsApi) index(ctx *z.Ctx) bool {
	aa.www.ServeHTTP(ctx.Writer, ctx.Request)
	return true
}
