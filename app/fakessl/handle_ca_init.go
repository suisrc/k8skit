package fakessl

import (
	"encoding/json"
	"fmt"
	"k8skit/app"
	"strings"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
	"github.com/suisrc/zgg/z/ze/tlsx"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (aa *FakeSslApi) caInit(zrc *z.Ctx) {
	cli := aa.K8sClient
	// ------------------------------------------------------------------------------------------
	co, err := z.ReadForm(zrc.Request, &SSLQueryCO{})
	if err != nil {
		zrc.JERR(err, 400)
		return
	}
	if co.Token == "" {
		zrc.JERR(&z.Result{ErrCode: "param-empty", Message: "token is empty"}, 400)
		return
	}
	if co.Key == "" {
		zrc.JERR(&z.Result{ErrCode: "param-empty", Message: "key is empty"}, 400)
		return
	}
	// ------------------------------------------------------------------------------------------
	config := tlsx.CertConfig{}
	if _, err := z.ReadBody(zrc.Request, &config); err != nil {
		message := fmt.Sprintf("init api, read body error: %s", err.Error())
		z.Println(message)
		zrc.JERR(&z.Result{ErrCode: "body-error", Message: message}, 400)
		return
	}
	k8sns := zc.GetNamespace()
	ikey := fmt.Sprintf("%s%s-%s", PK, co.Key, "data") // fkc-tst-data
	info, err := cli.CoreV1().Secrets(k8sns).Get(zrc.Ctx, ikey, metav1.GetOptions{})
	if err != nil {
		if !strings.HasSuffix(err.Error(), " not found") {
			message := fmt.Sprintf("init api, secret [%s] get error: %s", ikey, err.Error())
			z.Println(message)
			zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
			return
		}
		// 使用应用令牌， 进行验证，成功后，新建一个 INFO
		if co.Token != app.C.Token {
			message := fmt.Sprintf("init api, secret [%s] token error: no equal!", ikey)
			z.Println(message)
			zrc.JERR(&z.Result{ErrCode: "app-info-token", Message: message}, 500)
			return
		}
		co.Token = zc.GenStr("v", 32) // 生成一个新令牌，新建应用
		info = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: ikey}, Data: map[string][]byte{
			"token":  []byte(co.Token),
			"config": []byte(config.String()),
		}}
		info, err = cli.CoreV1().Secrets(k8sns).Create(zrc.Ctx, info, metav1.CreateOptions{})
		if err != nil {
			message := fmt.Sprintf("init api, secret [%s] create error: %s", ikey, err.Error())
			z.Println(message)
			zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
			return
		}
	}
	if tkn, ok := info.Data["token"]; ok && string(tkn) != co.Token { // 必须存在，不存在，不可访问
		message := fmt.Sprintf("init api, secret [%s].token error: no equal!", ikey)
		z.Println(message)
		zrc.JERR(&z.Result{ErrCode: "k8s-info-token", Message: message}, 500)
		return
	}
	// ------------------------------------------------------------------------------------------
	if cfgBts, ok := info.Data["config"]; ok { // 配置已经存在
		config2 := tlsx.CertConfig{}
		if err := json.Unmarshal(cfgBts, &config2); err != nil {
			message := fmt.Sprintf("init api, json unmarshal error: %s", err.Error())
			z.Println(message)
			zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
			return
		}
		// 合并配置, 更新配置
		if update := config2.Merge(config); update {
			info.Data["config"] = []byte(config2.String())
			info, err = cli.CoreV1().Secrets(k8sns).Update(zrc.Ctx, info, metav1.UpdateOptions{})
			if err != nil {
				message := fmt.Sprintf("init api, secret [%s] update error: %s", ikey, err.Error())
				z.Println(message)
				zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
				return
			}
		}
		config = config2
	} else { // 配置不存在
		info.Data["config"] = []byte(config.String())
		info, err = cli.CoreV1().Secrets(k8sns).Update(zrc.Ctx, info, metav1.UpdateOptions{})
		if err != nil {
			message := fmt.Sprintf("init api, secret [%s] create error: %s", ikey, err.Error())
			z.Println(message)
			zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
			return
		}
	}
	// ------------------------------------------------------------------------------------------
	if crt, ok := info.Data["ca.crt"]; ok {
		// 证书已经存在，立即返回
		zrc.JSON(&z.Result{Success: true, Data: string(crt)})
		return
	}
	// 证书不存在，需要重写构建证书
	ca, err := tlsx.CreateCA(config, co.CommonName)
	if err != nil {
		message := fmt.Sprintf("init api, create ca error: %s", err.Error())
		z.Println(message)
		zrc.JERR(&z.Result{ErrCode: "ca-create-err", Message: message}, 500)
		return
	}
	z.Println("init api, ca create success")
	info.Data["ca.crt"] = []byte(ca.Crt)
	info.Data["ca.key"] = []byte(ca.Key)
	// 求 ca.Key 的 md5 值
	ckey, _ := tlsx.HashMd5([]byte(ca.Key))
	info.Data["prefix"] = []byte(fmt.Sprintf("%s%s-%s-", PK, co.Key, ckey[:8]))
	_, err = cli.CoreV1().Secrets(k8sns).Update(zrc.Ctx, info, metav1.UpdateOptions{})
	if err != nil {
		message := fmt.Sprintf("init api, secret [%s] update error: %s", ikey, err.Error())
		z.Println(message)
		zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
		return
	}
	zrc.JSON(&z.Result{Success: true, Data: ca.Crt})
}
