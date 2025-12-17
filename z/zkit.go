package z

import (
	"cmp"
	_ "embed"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"strings"
	"unsafe"
)

//go:embed version
var verbyte []byte
var Version = strings.TrimSpace(string(verbyte))

// application and version
var AppName = os.Getenv("APP_NAME")
var AppVersion = strings.Join([]string{AppName, Version}, ":")

// ----------------------------------------------------------------------------

// GET http method
func GET(action string) string {
	return http.MethodGet + " " + action
}

// POST http method
func POST(action string) string {
	return http.MethodPost + " " + action
}

// 创建指针
func Ptr[T any](v T) *T {
	return &v
}

// 键值对
type Ref[K cmp.Ordered, T any] struct {
	Key K
	Val T
}

/**
 * 随机生成字符串， 0~f, 首字母不是 bb
 * @param bb 首字母
 */
func Str(bb string, ll int) string {
	str := []byte("0123456789abcdef")
	buf := make([]byte, ll-len(bb))
	for i := range buf {
		buf[i] = str[rand.Intn(len(str))]
	}
	return bb + string(buf)
}

// ----------------------------------------------------------------------------

/**
 * 可获取有权限字段
 */
func FieldValue(target any, field string) any {
	val := reflect.ValueOf(target)
	return val.Elem().FieldByName(field).Interface()
}

/**
 * 可设置字段值
 */
func FieldSetVal(target any, field string, value any) {
	val := reflect.ValueOf(target)
	val.Elem().FieldByName(field).Set(reflect.ValueOf(value))
}

/**
 * 获取 target 中每个字段的属性，注入和 value 属性的字段
 * 这只是一个演示的例子，实际开发中，请使用 SvcKit 模块
 */
func FieldInject(target any, value any, debug bool) bool {
	vType := reflect.TypeOf(value)
	tType := reflect.TypeOf(target).Elem()
	tElem := reflect.ValueOf(target).Elem()
	for i := 0; i < tType.NumField(); i++ {
		tField := tType.Field(i)
		tagVal := tField.Tag.Get("svckit")
		if tagVal != "type" && tagVal != "auto" {
			continue // `svckit:'type/auto'` 才可以通过类型注入
		}
		// 判断 vType 是否实现 tField.Type 的接口
		if tField.Type == vType || // 属性是一个接口，判断接口是否可以注入
			tField.Type.Kind() == reflect.Interface && vType.Implements(tField.Type) {
			// Printf("inject succ: %s", tField.Name)
			tElem.Field(i).Set(reflect.ValueOf(value))
			if debug {
				Printf("[_inject_]: [succ] %s.%s <- %s", tType, tField.Name, vType)
			}
			return true // 注入成功
		}
	}
	if debug {
		Printf("[_inject_]: [fail] %s not found field.(%s)", tType, vType)
	}
	return false
}

/**
 * 获取字段, 可夸包获取私有字段
 * 闭包原则，原则上不建议使用该方法，因为改方法是在破坏闭包原则
 */
func FieldValue_(target any, field string) any {
	val := reflect.ValueOf(target)
	vap := unsafe.Pointer(val.Elem().FieldByName(field).UnsafeAddr())
	return *(*any)(vap)
}

// ----------------------------------------------------------------------------
// ----------------------------------------------------------------------------

// 获取 traceID / 配置 traceID
func GetTraceID(request *http.Request) string {
	traceid := request.Header.Get("X-Request-Id")
	if traceid == "" {
		traceid = Str("r", 32) // 创建请求ID, 用于追踪
		request.Header.Set("X-Request-Id", traceid)
	}
	return traceid
}

// 获取 reqType / 配置 reqType
func GetReqType(server IServer, request *http.Request) string {
	reqtype := request.Header.Get("X-Request-Rt")
	if reqtype == "" {
		reqtype = server.GetConfig().B().ReqXrtd
		if reqtype != "" {
			request.Header.Set("X-Request-Rt", reqtype)
		}
	}
	return reqtype
}

// handler for core.HandleFunc to http.HandlerFunc
type handlerHF struct {
	sk SvcKit
	fn HandleFunc
}

func (aa handlerHF) handle(rw http.ResponseWriter, rr *http.Request) {
	ctx := NewCtx(aa.sk, rr, rw)
	defer ctx.Cancel()
	aa.fn(ctx)
}

// core.HandleFunc 转 http.HandlerFunc
func ToHttpFunc(svckit SvcKit, handle HandleFunc) http.HandlerFunc {
	return handlerHF{sk: svckit, fn: handle}.handle
}

// hander for multi func to one func
type handlerMF struct {
	hs []HandleFunc
}

func (aa handlerMF) handle(rc *Ctx) bool {
	for _, hh := range aa.hs {
		if hh(rc) {
			rc.Abort()
			return true
		}
	}
	return false
}

// merge multi func to one func
func MergeFunc(handles ...HandleFunc) HandleFunc {
	return handlerMF{hs: handles}.handle
}

// request token auth
func TokenAuth(token *string, handle HandleFunc) HandleFunc {
	// 需要验证令牌
	return func(ctx *Ctx) bool {
		if token == nil || *token == "" {
			return handle(ctx) // auth pass
		} else if ktn := ctx.Request.Header.Get("Authorization"); ktn == "Token "+*token {
			return handle(ctx) // auth succ
		}
		return ctx.JSON(&Result{ErrCode: "invalid-token", Message: "无效的令牌"})
	}
}
