package fakessl

import (
	"encoding/base64"
	"errors"
	"fmt"
	"k8skit/app"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func init() {
	z.Register("88-app.fakessl", func(zgg *z.Zgg) z.Closed {
		api := z.Inject(zgg.SvcKit, &FakeSslApi{})
		z.POST("api/ssl/v1/ca/init", api.caInit, zgg)
		z.GET("api/ssl/v1/ca", api.caGet, zgg)
		z.GET("api/ssl/v1/ca/txt", api.caTxt, zgg)
		z.GET("api/ssl/v1/ca/b64", api.caB64, zgg)
		z.GET("api/ssl/v1/cert", api.ceGet, zgg) // certificate
		z.GET("api/ssl/v1/ce", api.ceGet, zgg)
		return nil
	})
}

const PK = "fkc-"

type SSLQueryCO struct {
	Token      string   `form:"token"`
	Key        string   `form:"key"`
	Kind       int      `form:"kind"`
	CommonName string   `form:"cn"`
	Domains    []string `form:"domain"`
}

type FakeSslApi struct {
	K8sClient kubernetes.Interface `svckit:"k8sclient"`
}

func (aa *FakeSslApi) getCaSecret(zrc *z.Ctx) (*v1.Secret, int, error) {
	cli := aa.K8sClient
	// ---------------------------------------------------------------------------
	co, err := z.ReadForm(zrc.Request, &SSLQueryCO{})
	if err != nil {
		return nil, 400, err
	}
	if co.Key == "" {
		return nil, 400, &z.Result{ErrCode: "param-empty", Message: "key is empty"}
	}
	// ---------------------------------------------------------------------------
	k8sns := app.K8sNS()
	ikey := fmt.Sprintf("%s%s-%s", PK, co.Key, "data") // fkc-tst-data
	info, err := cli.CoreV1().Secrets(k8sns).Get(zrc.Ctx, ikey, metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("ca get api, secret [%s] get error: %s", ikey, err.Error())
		return nil, 500, errors.New(message)
	}
	_, ok := info.Data["ca.crt"]
	if !ok {
		message := fmt.Sprintf("ca get api, secret [%s] get error: ca.crt not found", ikey)
		return nil, 500, errors.New(message)
	}
	return info, 0, nil
}

// -------------------------------------------------------------------------------

func (aa *FakeSslApi) caGet(zrc *z.Ctx) {
	info, hss, err := aa.getCaSecret(zrc)
	if err != nil {
		zc.Println(err.Error())
		zrc.JERR(err, hss)
	} else {
		zrc.JSON(&z.Result{Success: true, Data: string(info.Data["ca.crt"])})
	}
}

func (aa *FakeSslApi) caTxt(zrc *z.Ctx) {
	info, hss, err := aa.getCaSecret(zrc)
	if err != nil {
		zc.Println(err.Error())
		zrc.JERR(err, hss)
	} else {
		zrc.TEXT(string(info.Data["ca.crt"]), 0)
	}
}

func (aa *FakeSslApi) caB64(zrc *z.Ctx) {
	info, hss, err := aa.getCaSecret(zrc)
	if err != nil {
		zc.Println(err.Error())
		zrc.JERR(err, hss)
	} else {
		b64 := base64.StdEncoding.EncodeToString(info.Data["ca.crt"])
		zrc.TEXT(b64, 0)
	}
}
