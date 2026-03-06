package k8s

import (
	"crypto/md5"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
)

var (
	C = struct {
		K8sSync Config
	}{}
)

type Config struct {
	Token string
	Debug bool     // true: 支持 Token 明文， 否则 Token 用来签名
	ExcNs []string // 忽略的命名空间
}

func init() {
	z.Config(&C)

	flag.StringVar(&C.K8sSync.Token, "kstoken", "", "k8s sync token")
	flag.BoolVar(&C.K8sSync.Debug, "ksdebug", false, "k8s sync debug")
	flag.Var(zc.NewStrArr(&C.K8sSync.ExcNs, []string{
		"kube-system",     //
		"kube-public",     //
		"kube-node-lease", //
		"kube-ksidecar",   //
		"kube-logs",       //
		"kube-reloader",   //
		"kube-rsync",      //
		"ingress-nginx",   //
		"metallb-system",  //
		"cert-manager",    //
		"base",            //
		"default",         //
		"devops",          //
		"kafka",           //
		"kuboard",         //
		"nats-dev",        //
		"nats-uat",        //
	}), "ksexcns", "k8s sync ignore namespace")

	z.Register("61-k8s.api", InitServe)
}

func CheckToken(zrc *z.Ctx) bool {
	if C.K8sSync.Token == "" {
		zrc.JERR(fmt.Errorf("no set token"), 401)
		return false
	}
	qry := zrc.Request.URL.Query()
	sig := qry.Get("sig")
	if sig == "" {
		zrc.JERR(fmt.Errorf("no signature"), 401)
		return false
	}
	if C.K8sSync.Debug && sig == C.K8sSync.Token {
		return true // 明文验证通过
	}
	// 进行密文验证校验
	if rnd := qry.Get("rand"); rnd == "" {
		zrc.JERR(fmt.Errorf("no random"), 401)
		return false
	} else if tim := qry.Get("time"); tim == "" {
		zrc.JERR(fmt.Errorf("no timestamp"), 401)
		return false
	} else if cim, err := strconv.Atoi(tim); err != nil {
		zrc.JERR(fmt.Errorf("invalid timestamp"), 401)
		return false
	} else if cim < (int(time.Now().Unix()) - 60) {
		zrc.JERR(fmt.Errorf("timestamp expired"), 401) // 60s内有效
		return false
	}
	raw := zrc.Request.RequestURI
	if idx := strings.Index(raw, "sig="); idx > 0 {
		raw = raw[:idx]
	}
	raw += "#" + C.K8sSync.Token
	// md5 签名
	hex1 := md5.Sum([]byte(raw))
	sig2 := z.HexStr(hex1[:])
	if sig != sig2 {
		zrc.JERR(fmt.Errorf("signature invalid"), 401)
		return false
	}
	return true
}
