package tls

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"k8skit/app"
	"strings"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
	"github.com/suisrc/zgg/ze/crt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	C = struct {
		SecretName string
	}{}
)

func init() {
	zc.Register(&C)
	flag.StringVar(&C.SecretName, "secretName", "", "sidecar fakessl secret name")

	z.Register("21-tlshttp", InitRegister)
}

func InitRegister(zgg *z.Zgg) z.Closed {
	if C.SecretName == "" {
		zc.Println("[_tlshttp]: SecretName is empty")
		return nil
	}
	// 注册 https 证书
	cli := zgg.SvcKit.Get("k8sclient").(kubernetes.Interface)

	// fkc-sidecar-data
	zc.Printf("[_tlshttp]: checker https cert: %s\n", C.SecretName)

	ctx := context.TODO()
	k8sns := app.K8sNS()
	ikey := C.SecretName
	info, err := cli.CoreV1().Secrets(k8sns).Get(ctx, C.SecretName, metav1.GetOptions{})
	if err != nil {
		if !strings.HasSuffix(err.Error(), " not found") {
			message := fmt.Sprintf("[_tlshttp]: secret [%s] get error: %s", ikey, err.Error())
			zgg.ServeStop(message) // 初始化失败，直接退出
			return nil
		}
		config := crt.CertConfig{"default": {
			Expiry: "20y",
			SubjectName: crt.SignSubject{
				Organization:     "default",
				OrganizationUnit: "default",
			},
		}}
		// 证书不存在，需要重写构建证书
		ca, err := crt.CreateCA(config, "default")
		if err != nil {
			message := fmt.Sprintf("[_tlshttp]: create ca error: %s", err.Error())
			zgg.ServeStop(message) // 初始化失败，直接退出
			return nil
		}
		ckey, _ := crt.HashMd5([]byte(ca.Key))
		pkey := strings.TrimSuffix(ikey, "-data") + "-" + ckey[:8]

		token := z.GenStr("v", 32) // 生成一个新令牌，新建应用
		info = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: ikey}, Data: map[string][]byte{
			"token":  []byte(token),
			"config": []byte(config.String()),
			"ca.crt": []byte(ca.Crt),
			"ca.key": []byte(ca.Key),
			"prefix": []byte(pkey),
		}}
		info, err = cli.CoreV1().Secrets(k8sns).Create(ctx, info, metav1.CreateOptions{})
		if err != nil {
			message := fmt.Sprintf("[_tlshttp]: secret [%s] create error: %s", ikey, err.Error())
			zgg.ServeStop(message) // 初始化失败，直接退出
			return nil
		}
	}
	if token, ok := info.Data["token"]; !ok {
		message := fmt.Sprintf("[_tlshttp]: secret [%s.token] not found", ikey)
		zgg.ServeStop(message) // 初始化失败，直接退出
		return nil
	} else {
		app.C.Token = string(token)
		zc.Printf("[_tlshttp]: secret [%s] token: %s", ikey, string(token))
	}

	config := crt.TLSAutoConfig{}
	if cfgBts, ok := info.Data["config"]; !ok {
		message := fmt.Sprintf("[_tlshttp]: secret [%s.config] not found", ikey)
		zgg.ServeStop(message) // 初始化失败，直接退出
		return nil
	} else if err := json.Unmarshal(cfgBts, &config.CertConf); err != nil {
		message := fmt.Sprintf("[_tlshttp]: json unmarshal error: %s", err.Error())
		zgg.ServeStop(message) // 初始化失败，直接退出
		return nil
	}
	if crtBts, ok := info.Data["ca.crt"]; !ok {
		message := fmt.Sprintf("[_tlshttp]: secret [%s.(ca.crt)] not found", ikey)
		zgg.ServeStop(message) // 初始化失败，直接退出
		return nil
	} else {
		config.CaCrtBts = crtBts
	}
	if keyBts, ok := info.Data["ca.key"]; !ok {
		message := fmt.Sprintf("[_tlshttp]: secret [%s.(ca.key)] not found", ikey)
		zgg.ServeStop(message) // 初始化失败，直接退出
		return nil
	} else {
		config.CaKeyBts = keyBts
	}

	cfg := &tls.Config{}
	cfg.GetCertificate = (&config).GetCertificate
	zgg.TLSConf = cfg

	return nil
}
