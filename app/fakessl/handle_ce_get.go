package fakessl

import (
	"encoding/json"
	"fmt"
	"kube-sidecar/app"
	"kube-sidecar/z"
	"net"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (aa *FakeSslApi) ceGet(zrc *z.Ctx) bool {
	cli := aa.K8sClient
	// ------------------------------------------------------------------------------------------
	co, err := z.ByTag(&SSLQueryCO{}, zrc.Request.URL.Query(), "form")
	if err != nil {
		return zrc.JERR(err, 400)
	}
	if co.Token == "" {
		return zrc.JERR(&z.Result{ErrCode: "param-empty", Message: "token is empty"}, 400)
	}
	if co.Key == "" {
		return zrc.JERR(&z.Result{ErrCode: "param-empty", Message: "key is empty"}, 400)
	}
	if len(co.Domains) == 0 {
		return zrc.JERR(&z.Result{ErrCode: "param-empty", Message: "domains is empty"}, 400)
	}
	// ------------------------------------------------------------------------------------------
	k8sns := app.K8sNs()
	ikey := fmt.Sprintf("%s%s-%s", PK, co.Key, "data") // fkc-tst-data
	info, err := cli.CoreV1().Secrets(k8sns).Get(zrc.Ctx, ikey, metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("ce get api, secret [%s] get error: %s", ikey, err.Error())
		return zrc.JERR(&z.Result{ErrCode: "k8s-info-get", Message: message}, 500)
	}
	if tkn, ok := info.Data["token"]; ok && string(tkn) != co.Token { // 必须存在，不存在，不可访问
		message := fmt.Sprintf("ce get api, secret [%s].token error: no equal!", ikey)
		return zrc.JERR(&z.Result{ErrCode: "k8s-info-token", Message: message}, 500)
	}
	// ------------------------------------------------------------------------------------------
	secretKey := fmt.Sprintf("%s%s-", PK, co.Key)
	if bts, ok := info.Data["prefix"]; ok {
		secretKey = string(bts)
	}
	if len(co.Domains) == 1 {
		secretKey += co.Domains[0]
	} else {
		sort.Strings(co.Domains)
		hash, _ := z.HashMd5([]byte(strings.Join(co.Domains, ",")))
		secretKey += hash
	}
	// ------------------------------------------------------------------------------------------
	isUpdate := false
	domain, err := cli.CoreV1().Secrets(k8sns).Get(zrc.Ctx, secretKey, metav1.GetOptions{})
	if err == nil {
		if ok, _ := z.IsPemExpired(string(domain.Data["pem.crt"])); !ok {
			return zrc.JSON(&z.Result{Success: true, Data: z.HA{
				"crt": string(domain.Data["pem.crt"]),
				"key": string(domain.Data["pem.key"]),
			}})
		}
		isUpdate = true
		// 证书出现问题或者过期
	}
	if co.Kind != 1 {
		return zrc.JERR(&z.Result{ErrCode: "param-error", Message: "kind is error"}, 400)
	}
	// domain 对应的cert 不存在, 重新生成 cert
	dns := []string{} // 域名
	ips := []net.IP{}
	// reg, _ := regexp.Compile(`^(\d{1,3}\.){3}\d{1,3}$`)
	for _, domain := range co.Domains {
		// 正则表达式匹配IP， 暂时支持ipv4
		if ip := net.ParseIP(domain); ip != nil {
			ips = append(ips, ip)
		} else {
			dns = append(dns, domain) // 域名
		}
	}
	caBts, ok0 := info.Data["config"]
	caCrt, ok1 := info.Data["ca.crt"]
	caKey, ok2 := info.Data["ca.key"]
	if !ok0 || !ok1 || !ok2 {
		return zrc.JERR(&z.Result{ErrCode: "k8s-info-error", Message: "config is error, CA证书不存在"}, 500)
	}
	config := &z.CertConfig{}
	if err := json.Unmarshal(caBts, &config); err != nil {
		message := fmt.Sprintf("ce get api, json unmarshal config error: %s", err.Error())
		return zrc.JERR(&z.Result{ErrCode: "k8s-info-error", Message: message}, 500)

	}
	sub, err := z.CreateCE(config, co.CommonName, co.Profile, 0, dns, ips, caCrt, caKey)
	if err != nil {
		message := fmt.Sprintf("ce get api, create cert error: %s", err.Error())
		return zrc.JERR(&z.Result{ErrCode: "k8s-info-error", Message: message}, 500)
	}
	// 存储 k8s secret
	domain = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretKey}, StringData: map[string]string{
		"pem.crt": sub.Crt,
		"pem.key": sub.Key,
		"domains": strings.Join(co.Domains, ","),
	}}
	if isUpdate {
		if _, err := cli.CoreV1().Secrets(k8sns).Update(zrc.Ctx, domain, metav1.UpdateOptions{}); err != nil {
			message := fmt.Sprintf("ce get api, secret [%s] update error: %s", secretKey, err.Error())
			return zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
		}
	} else {
		if _, err := cli.CoreV1().Secrets(k8sns).Create(zrc.Ctx, domain, metav1.CreateOptions{}); err != nil {
			message := fmt.Sprintf("ce get api, secret [%s] create error: %s", secretKey, err.Error())
			return zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
		}
	}
	// ------------------------------------------------------------------------------------------
	return zrc.JSON(&z.Result{Success: true, Data: z.HA{
		"crt": domain.StringData["pem.crt"],
		"key": domain.StringData["pem.key"],
	}})
}
