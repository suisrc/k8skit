package z

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// 创建上下文函数
func NewCtx(svckit SvcKit, request *http.Request, writer http.ResponseWriter) *Ctx {
	action := request.URL.Query().Get("action")
	if action == "" {
		rpath := request.URL.Path
		if len(rpath) > 0 {
			rpath = rpath[1:] // 删除前缀 '/'
		}
		action = rpath
	}
	ctx := &Ctx{SvcKit: svckit, Action: action, Cache: HA{}, Request: request, Writer: writer}
	ctx.Ctx, ctx.Cancel = context.WithCancel(context.Background())
	ctx.TraceID = GetTraceID(request)
	ctx.ReqType = GetReqType(svckit.Srv(), request)
	return ctx
}

// ----------------------------------------------------------------------------

// 请求上下文内容
type Ctx struct {
	Ctx context.Context
	// cancel func
	Cancel context.CancelFunc
	// All Module
	SvcKit SvcKit
	// request action
	Action string
	// request share data
	Cache HA
	// request
	Request *http.Request
	// response
	Writer http.ResponseWriter
	// trace id
	TraceID string
	// X-Request-Rt
	ReqType string
	// flag action abort
	_abort bool
}

// 用于标记提前结束，不是强制的
func (ctx *Ctx) Abort() {
	ctx._abort = true
}

func (ctx *Ctx) IsAbort() bool {
	return ctx._abort
}

// 已 JSON 格式写出响应
func (ctx *Ctx) JSON(err error) bool {
	ctx._abort = true // rc.Abort()
	// 注意，推荐使用 JSON(rc, rs), 这里只是为了简化效用逻辑
	switch err := err.(type) {
	case *Result:
		return JSON(ctx, err)
	default:
		return JSON(ctx, &Result{ErrCode: "unknow-error", Message: err.Error()})
	}
}

// 已 HTML 模板格式写出响应
func (ctx *Ctx) HTML(tpl string, res any, hss int) bool {
	ctx._abort = true
	ctx.Writer.Header().Set("Api-Version", AppVersion)
	if ctx.TraceID != "" {
		ctx.Writer.Header().Set("X-Request-Id", ctx.TraceID)
	}
	if hss > 0 {
		ctx.Writer.WriteHeader(hss)
	}
	return HTML0(ctx.SvcKit.Srv(), ctx.Request, ctx.Writer, res, tpl)
}

// 已 TEXT 模板格式写出响应
func (ctx *Ctx) TEXT(txt string, hss int) bool {
	ctx._abort = true
	ctx.Writer.Header().Set("Api-Version", AppVersion)
	if ctx.TraceID != "" {
		ctx.Writer.Header().Set("X-Request-Id", ctx.TraceID)
	}
	ctx.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if hss > 0 {
		ctx.Writer.WriteHeader(hss) // 最后写状态码头
	}
	ctx.Writer.Write([]byte(txt))
	return true
}

// 已 JSON 错误格式写出响应
func (ctx *Ctx) JERR(err error, hss int) bool {
	ctx._abort = true // rc.Abort()
	// 注意，推荐使用 JSON(rc, rs), 这里只是为了简化效用逻辑
	var res *Result
	switch err := err.(type) {
	case *Result:
		res = err
	default:
		res = &Result{ErrCode: "unknow-error", Message: err.Error()}
	}
	if hss > 0 {
		res.Status = hss
	}
	return JSON(ctx, res)
}

// ----------------------------------------------------------------------------
// ----------------------------------------------------------------------------

// 请求数据
func ReadBody[T any](rr *http.Request, rb T) (T, error) {
	return rb, json.NewDecoder(rr.Body).Decode(rb)
}

// 请求结构体
type RaData struct {
	Atyp string `json:"type"`
	Data string `json:"data"`
}

// 请求数据
func ReadData(rr *http.Request) (*RaData, error) {
	return ReadBody(rr, &RaData{})
}

// ----------------------------------------------------------------------------
// ----------------------------------------------------------------------------

// 定义响应结构体
type Result struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	ErrCode string `json:"errcode,omitempty"`
	Message string `json:"message,omitempty"`
	TraceID string `json:"traceid,omitempty"`

	Ctx    *Ctx   `json:"-"`
	Status int    `json:"-"`
	Header HS     `json:"-"`
	TplKey string `json:"-"`
}

func (aa *Result) Error() string {
	return fmt.Sprintf("[%v], %s, %s", aa.Success, aa.ErrCode, aa.Message)
}

// ----------------------------------------------------------------------------
// ----------------------------------------------------------------------------

// 响应 JSON 结果, 这是一个套娃，
func JSON(ctx *Ctx, res *Result) bool {
	res.Ctx = ctx
	// 响应基础头部
	ctx.Writer.Header().Set("Api-Version", AppVersion)
	// TraceID 可能不存在，如果不是 '' 则 PASS
	if res.TraceID == "" {
		res.TraceID = ctx.TraceID
	}
	if res.TraceID != "" {
		ctx.Writer.Header().Set("X-Request-Id", res.TraceID)
	}
	// 响应其他头部
	if ctx.Request.Header != nil { // 设置响应头
		for k, v := range res.Header {
			ctx.Writer.Header().Set(k, v)
		}
	}
	// 响应结果
	switch ctx.ReqType {
	case "2":
		return JSON2(ctx.Request, ctx.Writer, res)
	case "3":
		return HTML3(ctx.Request, ctx.Writer, res)
	default:
		return JSON0(ctx.Request, ctx.Writer, res)
	}
}

// 响应 JSON 结果: content-type http-status json-data
func JSON0(rr *http.Request, rw http.ResponseWriter, rs *Result) bool {
	// 响应结果
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	if rs.Status > 0 {
		rw.WriteHeader(rs.Status)
	}
	json.NewEncoder(rw).Encode(rs)
	return true
}

// 以 '2' 形式格式化, 响应 JSON 结果: content-type http-status json-data
func JSON2(rr *http.Request, rw http.ResponseWriter, rs *Result) bool {
	// 转换结构
	ha := HA{"success": rs.Success}
	if rs.Data != nil {
		ha["data"] = rs.Data
	}
	if rs.ErrCode != "" {
		ha["errorCode"] = rs.ErrCode
		ha["errorMessage"] = rs.Message
	}
	if rs.TraceID != "" {
		ha["traceId"] = rs.TraceID
	}
	// 响应结果
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	if rs.Status > 0 {
		rw.WriteHeader(rs.Status)
	}
	json.NewEncoder(rw).Encode(ha)
	return true
}

// 以 '3' 形式格式化, 选择模版，响应 HTML 模板结果: content-type http-status html-data
func HTML3(rr *http.Request, rw http.ResponseWriter, rs *Result) bool {
	if rs.Ctx == nil {
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rw.Write([]byte("template render error: request content not found"))
		return true
	}
	tmpl := rs.TplKey
	if tmpl == "" {
		if rs.Success {
			tmpl = "success.html"
		} else {
			tmpl = "error.html"
		}
	}
	if rs.Status > 0 {
		rw.WriteHeader(rs.Status)
	}
	return HTML0(rs.Ctx.SvcKit.Srv(), rr, rw, rs, tmpl)
}

// 响应 HTML 模板结果: content-type http-status html-data
func HTML0(sv IServer, rr *http.Request, rw http.ResponseWriter, rs any, tp string) bool {
	// 响应结果
	if sv == nil {
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rw.Write([]byte("template render error: server not found"))
	} else {
		err := sv.GetTplKit().Render(rw, tp, rs)
		if err != nil {
			rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
			rw.Write([]byte("template render error: " + err.Error()))
		} else {
			rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
	}
	return true
}
