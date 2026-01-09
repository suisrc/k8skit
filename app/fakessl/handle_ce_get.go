package fakessl

import (
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/ze/crt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (aa *FakeSslApi) ceGet(zrc *z.Ctx) {
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
	if len(co.Domains) == 0 {
		zrc.JERR(&z.Result{ErrCode: "param-empty", Message: "domains is empty"}, 400)
		return
	}
	// ------------------------------------------------------------------------------------------
	k8sns := z.GetNamespace()
	ikey := fmt.Sprintf("%s%s-%s", PK, co.Key, "data") // fkc-tst-data
	info, err := cli.CoreV1().Secrets(k8sns).Get(zrc.Ctx, ikey, metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("ce get api, secret [%s] get error: %s", ikey, err.Error())
		zrc.JERR(&z.Result{ErrCode: "k8s-info-get", Message: message}, 500)
		return
	}
	if tkn, ok := info.Data["token"]; ok && string(tkn) != co.Token { // 必须存在，不存在，不可访问
		message := fmt.Sprintf("ce get api, secret [%s].token error: no equal!", ikey)
		zrc.JERR(&z.Result{ErrCode: "k8s-info-token", Message: message}, 500)
		return
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
		hash, _ := crt.HashMd5([]byte(strings.Join(co.Domains, ",")))
		secretKey += hash
	}
	// ------------------------------------------------------------------------------------------
	isUpdate := false
	domain, err := cli.CoreV1().Secrets(k8sns).Get(zrc.Ctx, secretKey, metav1.GetOptions{})
	if co.Kind != 3 && err == nil {
		if ok, exp, _ := crt.IsPemExpired(string(domain.Data["pem.crt"])); !ok {
			zrc.JSON(&z.Result{Success: true, Data: z.HA{
				"crt": string(domain.Data["pem.crt"]),
				"key": string(domain.Data["pem.key"]),
			}})
			return
		} else {
			z.Printf("ce get api, secret [%s] expired: %v", secretKey, exp)
		}
		isUpdate = true
		// 证书出现问题或者过期
	} else if err == nil {
		isUpdate = true
	}
	if co.Kind != 1 && co.Kind != 3 {
		zrc.JERR(&z.Result{ErrCode: "param-error", Message: "kind is error"}, 400)
		return
	}
	z.Printf("ce get api, secret [%s] create/update: %d", secretKey, co.Kind)
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
		zrc.JERR(&z.Result{ErrCode: "k8s-info-error", Message: "config is error, CA证书不存在"}, 500)
		return
	}
	config := crt.CertConfig{}
	if err := json.Unmarshal(caBts, &config); err != nil {
		message := fmt.Sprintf("ce get api, json unmarshal config error: %s", err.Error())
		zrc.JERR(&z.Result{ErrCode: "k8s-info-error", Message: message}, 500)
		return
	}
	sub, err := crt.CreateCE(config, co.CommonName, dns, ips, caCrt, caKey)
	if err != nil {
		message := fmt.Sprintf("ce get api, create cert error: %s", err.Error())
		zrc.JERR(&z.Result{ErrCode: "k8s-info-error", Message: message}, 500)
		return
	}
	// z.Println(sub.Crt)
	// 存储 k8s secret
	domain = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretKey}, Data: map[string][]byte{
		"pem.crt": []byte(sub.Crt),
		"pem.key": []byte(sub.Key),
		"domains": []byte(strings.Join(co.Domains, ",")),
	}}
	if isUpdate {
		if _, err := cli.CoreV1().Secrets(k8sns).Update(zrc.Ctx, domain, metav1.UpdateOptions{}); err != nil {
			message := fmt.Sprintf("ce get api, secret [%s] update error: %s", secretKey, err.Error())
			zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
			return
		}
	} else {
		if _, err := cli.CoreV1().Secrets(k8sns).Create(zrc.Ctx, domain, metav1.CreateOptions{}); err != nil {
			message := fmt.Sprintf("ce get api, secret [%s] create error: %s", secretKey, err.Error())
			zrc.JERR(&z.Result{ErrCode: "k8s-secret-err", Message: message}, 500)
			return
		}
	}
	// ------------------------------------------------------------------------------------------
	zrc.JSON(&z.Result{Success: true, Data: z.HA{
		"crt": string(domain.Data["pem.crt"]),
		"key": string(domain.Data["pem.key"]),
	}})
}
