package iam

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"k8skit/app"
	"k8skit/app/zdb"
	"net/http"
	"strings"
	"time"

	"github.com/suisrc/zgg/z"
)

// 判断是否登录
func (s *IamServeApi) Authx(zrc *z.Ctx) {
	// 处理 Cookie
	for _, cookie := range zrc.Request.Cookies() {
		zrc.Caches["cookie_"+cookie.Name] = cookie
	}
	authx := zrc.Request.Header.Get("Authorization")
	if z.IsDebug() && authx != "" {
		z.Println("[authori~]: Authorization, ", authx)
	}
	realm := "fmesui"
	nonces := ""
	nerror := "Unauthorized"

	if cookie, ok := zrc.Caches["cookie_kot"].(*http.Cookie); !ok || cookie.Value == "" {
		// 没有 cookie， 要求用户重新登录， 直接返回404
		cval := z.GenStr("kot.", 44)
		http.SetCookie(zrc.Writer, &http.Cookie{
			Name:     "kot",
			Value:    cval,
			Path:     "/",
			HttpOnly: true,                    // 防止 XSS 攻击
			Secure:   true,                    // 仅通过 HTTPS 传输
			SameSite: http.SameSiteStrictMode, // 防止 CSRF 攻击
			// MaxAge: -1: 会话 Cookie，浏览器关闭时自动删除, 0: 立即过期
		})
		user := &app.User{}
		user.Nonces = z.GenStr("", 16)
		user.ExpireAt = time.Now().Unix() + 600 // 有效期10分钟
		s.CAC.SetX(zrc.Ctx, "user_"+cval, user, 600*time.Second)
		// 要求用户完成登录
		nonces = user.Nonces
	} else if user, ok, _ := s.CAC.GetX(zrc.Ctx, "user_"+cookie.Value); !ok {
		// 令牌无效，要求用户重新登录
		user := &app.User{}
		user.Nonces = z.GenStr("", 16)
		user.ExpireAt = time.Now().Unix() + 600 // 有效期10分钟
		s.CAC.SetX(zrc.Ctx, "user_"+cookie.Value, user, 600*time.Second)
		// 要求用户完成登录
		nonces = user.Nonces
		nerror = "Login timeout, please login again"
	} else if user, ok := user.(*app.User); !ok {
		// 系统内部异常， 用户状态不对，重建用户信息
		user := &app.User{}
		user.Nonces = z.GenStr("", 16)
		user.ExpireAt = time.Now().Unix() + 600 // 有效期10分钟
		s.CAC.SetX(zrc.Ctx, "user_"+cookie.Value, user, 600*time.Second)
		// 要求用户完成登录
		nonces = user.Nonces
		nerror = "Invalid cache, please login again"
	} else if user.IsLogin {
		// ==================================================
		// 用户已经登录, 用户已经登录, 用户已经登录
		// ==================================================
		zrc.Caches["user"] = user // 请求上下文中缓存登录人信息
	} else if user.ExpireAt < time.Now().Unix() {
		// 令牌已过期，要求用户重新登录, 防止 nonce 被碰撞
		user.Nonces = z.GenStr("", 16)
		user.ExpireAt = time.Now().Unix() + 600 // 有效期10分钟
		s.CAC.SetX(zrc.Ctx, "user_"+cookie.Value, user, 600*time.Second)
		// 要求用户完成登录
		nonces = user.Nonces
		nerror = "Login timeout, please login again"
	} else if authz := zrc.Request.Header.Get("Authorization"); authz == "" {
		// 要求用户完成登录， 没有验证信息完成登录
		nonces = user.Nonces
		nerror = "Login info is empty"
	} else if !strings.HasPrefix(authx, "Digest ") {
		// 认证方式不正确，需要 Digest 认证方式
		nonces = user.Nonces
		nerror = "Login type error, please login again"
	} else if auths := strings.Split(authx[7:], ", "); len(auths) < 5 {
		// 认证信息格式错误, Digest xxx
		nonces = user.Nonces
		nerror = "Login info format error, please login again"
	} else {
		z.Println("[authori~]:", auths)
		username, realmstr, noncestr, paramuri, response := "", "", "", "", ""
		qop, pnc, pcc := "", "", ""
		for _, data := range auths {
			switch {
			case strings.HasPrefix(data, "username="):
				username = strings.Trim(data[9:], "\"")
			case strings.HasPrefix(data, "realm="):
				realmstr = strings.Trim(data[6:], "\"")
			case strings.HasPrefix(data, "nonce="):
				noncestr = strings.Trim(data[6:], "\"")
			case strings.HasPrefix(data, "uri="):
				paramuri = strings.Trim(data[4:], "\"")
			case strings.HasPrefix(data, "response="):
				response = strings.Trim(data[9:], "\"")
			case strings.HasPrefix(data, "qop="):
				qop = strings.Trim(data[4:], "\"")
			case strings.HasPrefix(data, "nc="):
				pnc = strings.Trim(data[3:], "\"")
			case strings.HasPrefix(data, "cnonce="):
				pcc = strings.Trim(data[7:], "\"")
			}
		}
		if pnc != "00000001" {
			// 只处理首次请求，之后的请求不在验证范围之内
			nonces = user.Nonces
			nerror = "Login nc is error, please login again"
		} else if realmstr == "" || realmstr != realm {
			nonces = user.Nonces
			nerror = "Login realm is error, please login again"
		} else if noncestr == "" || noncestr != user.Nonces {
			nonces = user.Nonces
			nerror = "Login nonce is error, please login again"
		} else if paramuri == "" || paramuri != zrc.Request.RequestURI {
			nonces = user.Nonces
			nerror = "Login uri is error, please login again"
		} else if username == "" || response == "" {
			nonces = user.Nonces
			nerror = "Login username/passowrd is error, please login again"
		} else {
			// 进行用户验证
			if auth, err := s.FindUser(username); err != nil {
				z.Println("[authori~]: found user error,", username, err.Error())
				nonces = user.Nonces
				nerror = "Login username/passowrd is error, please login again"
			} else if auth == nil {
				z.Println("[authori~]: not found username,", username)
				nonces = user.Nonces
				nerror = "Login username/passowrd is error, please login again"
			} else {
				if z.IsDebug() {
					z.Println("[authori~]: hash content,", fmt.Sprintf("%s:%s:%s|%s:%s:%s:%s|%s:%s", //
						auth.AppKey.String, realm, "***", //
						user.Nonces, pnc, pcc, qop, //
						zrc.Request.Method, zrc.Request.RequestURI))
				}
				expected := ""
				// 计算 hash 值
				// ha1 = md5(username + ":" + realm + ":" + password)
				// ha2 = md5(method + ":" + uri)
				ha1 := md5.Sum(fmt.Appendf(nil, "%s:%s:%s", auth.AppKey.String, realm, auth.Secret.String))
				ha2 := md5.Sum(fmt.Appendf(nil, "%s:%s", zrc.Request.Method, zrc.Request.RequestURI))
				// // ha3 = md5(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2)) RFC 2617 规范
				// ha3 := md5.Sum(fmt.Appendf(nil, "%s:%s:%s", z.HexStr(ha1[:]), user.Nonces, z.HexStr(ha2[:])))
				// expected = z.HexStr(ha3[:])
				// ha3 = md5(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2)) RFC 7616 规范
				ha3 := md5.Sum(fmt.Appendf(nil, "%s:%s:%s:%s:%s:%s", z.HexStr(ha1[:]), user.Nonces, pnc, pcc, qop, z.HexStr(ha2[:])))
				expected = z.HexStr(ha3[:])
				// 验证 hash 值
				if response != expected {
					if z.IsDebug() {
						z.Println("[authori~]: check hash error,", expected, "<->", response)
					}
					nonces = user.Nonces
					nerror = "Login username/passowrd is error, please login again"
				} else {
					user.Session = cookie.Value
					user.IsLogin = true
					// user.Nonces = ""
					user.ExpireAt = -1
					s.CAC.SetX(zrc.Ctx, "user_"+cookie.Value, user, 7200*time.Second)
					zrc.Writer.Header().Set("WWW-Authenticate", "Clear")
					zrc.JSON(&z.Result{Success: true, Data: "reload", ErrShow: 8, Status: 401})
					zrc.Abort()
				}
			}
		}
	}
	if nonces != "" {
		if z.IsDebug() {
			z.Println("[authori~]: relogin,", nerror)
		}
		zrc.Writer.Header().Set("WWW-Authenticate", fmt.Sprintf(`Digest realm="%s", nonce="%s", qop="auth"`, realm, nonces))
		http.Error(zrc.Writer, nerror, http.StatusUnauthorized)
		zrc.Abort()
	}
	// next 处理
}

// 获取登录人信息
func (s *IamServeApi) FindUser(username string) (*zdb.AuthzDO, error) {
	return &zdb.AuthzDO{
		Name:   sql.NullString{String: username, Valid: true},
		AppKey: sql.NullString{String: username, Valid: true},
		Secret: sql.NullString{String: "123", Valid: true},
	}, nil
}
