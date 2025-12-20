package fakessl

import (
	"encoding/json"
	"fmt"
	"kube-sidecar/app"
	"strings"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/ze/crt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func (aa *FakeSslApi) caInit(zrc *z.Ctx) bool {
	cli := aa.K8sClient
	// ------------------------------------------------------------------------------------------
	co, err := z.ReadForm(zrc.Request, &SSLQueryCO{})
	if err != nil {
		return zrc.JERR(err, 400)
	}
	if co.Token == "" {
		return zrc.JERR(&z.Result{ErrCode: "param-empty", Message: "token is empty"}, 400)
	}
	if co.Key == "" {
		return zrc.JERR(&z.Result{ErrCode: "param-empty", Message: "key is empty"}, 400)
	}
	// ------------------------------------------------------------------------------------------
	config, err := z.ReadBody(zrc.Request, &crt.CertConfig{})
	if err != nil {
		message := fmt.Sprintf("init api, read body error: %s", err.Error())
		klog.Info(message)
		return zrc.JERR(&z.Result{ErrCode: "body-error", Message: message}, 400)
	}
	k8sns := app.K8sNs()
	ikey := fmt.Sprintf("%s%s-%s", PK, co.Key, "data") // fkc-tst-data
	info, err := cli.CoreV1().Secrets(k8sns).Get(zrc.Ctx, ikey, metav1.GetOptions{})
	if err != nil {
		if strings.HasSuffix(err.Error(), " not found") {
			// 使用应用令牌， 进行验证，成功后，新建一个 INFO
			if co.Token != app.C.Token {
				message := fmt.Sprintf("init api, secret [%s] token error: no equal!", ikey)
				klog.Info(message)
				return zrc.JERR(&z.Result{ErrCode: "app-info-token", Message: message}, 500)
			}
			co.Token = z.Str("v", 32) // 生成一个新令牌，新建应用
			info = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: ikey}, Data: map[string][]byte{
				"token":  []byte(co.Token),
				"config": []byte(config.String()),
			}}
			info, err = cli.CoreV1().Secrets(k8sns).Create(zrc.Ctx, info, metav1.CreateOptions{})
			if err != nil {
				message := fmt.Sprintf("init api, secret [%s] create error: %s", ikey, err.Error())
				klog.Info(message)
				return zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
			}
		} else {
			message := fmt.Sprintf("init api, secret [%s] get error: %s", ikey, err.Error())
			klog.Info(message)
			return zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
		}
	}
	if tkn, ok := info.Data["token"]; ok && string(tkn) != co.Token { // 必须存在，不存在，不可访问
		message := fmt.Sprintf("init api, secret [%s].token error: no equal!", ikey)
		klog.Info(message)
		return zrc.JERR(&z.Result{ErrCode: "k8s-info-token", Message: message}, 500)
	}
	// ------------------------------------------------------------------------------------------
	if cfgBts, ok := info.Data["config"]; ok { // 配置已经存在
		config2 := &crt.CertConfig{}
		if err := json.Unmarshal(cfgBts, config2); err != nil {
			message := fmt.Sprintf("init api, json unmarshal error: %s", err.Error())
			klog.Info(message)
			return zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
		}
		// 合并配置, 更新配置
		if update := config2.Merge(config); update {
			info.Data["config"] = []byte(config2.String())
			info, err = cli.CoreV1().Secrets(k8sns).Update(zrc.Ctx, info, metav1.UpdateOptions{})
			if err != nil {
				message := fmt.Sprintf("init api, secret [%s] update error: %s", ikey, err.Error())
				klog.Info(message)
				return zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
			}
		}
		config = config2
	} else { // 配置不存在
		info.Data["config"] = []byte(config.String())
		info, err = cli.CoreV1().Secrets(k8sns).Update(zrc.Ctx, info, metav1.UpdateOptions{})
		if err != nil {
			message := fmt.Sprintf("init api, secret [%s] create error: %s", ikey, err.Error())
			klog.Info(message)
			return zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
		}
	}
	// ------------------------------------------------------------------------------------------
	if crt, ok := info.Data["ca.crt"]; ok {
		// 证书已经存在，立即返回
		return zrc.JSON(&z.Result{Success: true, Data: string(crt)})
	}
	// 证书不存在，需要重写构建证书
	ca, err := crt.CreateCA(config)
	if err != nil {
		message := fmt.Sprintf("init api, create ca error: %s", err.Error())
		klog.Info(message)
		return zrc.JERR(&z.Result{ErrCode: "ca-create-err", Message: message}, 500)
	}
	klog.Info("init api, ca create success")
	info.Data["ca.crt"] = []byte(ca.Crt)
	info.Data["ca.key"] = []byte(ca.Key)
	// 求 ca.Key 的 md5 值
	ckey, _ := crt.HashMd5([]byte(ca.Key))
	info.Data["prefix"] = []byte(fmt.Sprintf("%s%s-%s-", PK, co.Key, ckey[:8]))
	_, err = cli.CoreV1().Secrets(k8sns).Update(zrc.Ctx, info, metav1.UpdateOptions{})
	if err != nil {
		message := fmt.Sprintf("init api, secret [%s] update error: %s", ikey, err.Error())
		klog.Info(message)
		return zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
	}
	return zrc.JSON(&z.Result{Success: true, Data: ca.Crt})
}
