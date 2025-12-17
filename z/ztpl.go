package z

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
)

var (
	ErrTplNotFound = errors.New("tpl not found")
)

type Tpl struct {
	Key string             // 模版编码
	Tpl *template.Template // 模版
	Err error              // 加载模版的异常
	Txt string             // 模版原始内容
	// once sync.Once      // 模版加载锁
}

type TplKit interface {
	Get(key string) *Tpl
	Render(wr io.Writer, name string, data any) error
	Load(key string, str string) *Tpl
	Preload(dir string, debug bool) error
}

// ----------------------------------------------------------------------------
// ----------------------------------------------------------------------------

type TplKit0 struct {
	tpls map[string]*Tpl // 所有模版集合
	lock sync.RWMutex    // 读写锁

	FuncMap template.FuncMap // 支持链式调用
}

func NewTplKit() TplKit {
	return &TplKit0{
		tpls: make(map[string]*Tpl),
	}
}

func (aa *TplKit0) Get(key string) *Tpl {
	aa.lock.RLock()
	defer aa.lock.RUnlock()
	return aa.tpls[key]
}

func (tk *TplKit0) Render(wr io.Writer, name string, data any) error {
	tpl := tk.Get(name)
	if tpl == nil {
		return ErrTplNotFound
	} else if tpl.Err != nil {
		return tpl.Err
	}
	return tpl.Tpl.Execute(wr, data)
}

func (aa *TplKit0) Load(key string, str string) *Tpl {
	aa.lock.Lock()
	defer aa.lock.Unlock()
	if tpl, ok := aa.tpls[key]; ok {
		return tpl
	}
	tpl := &Tpl{}
	tpl.Key = key
	tpl.Txt = str
	tpl.Tpl, tpl.Err = template.New(tpl.Key).Parse(tpl.Txt)
	if tpl.Err == nil {
		tpl.Tpl.Funcs(aa.FuncMap)
	}
	aa.tpls[tpl.Key] = tpl
	return tpl
}

func (aa *TplKit0) Preload(dir string, debug bool) error {
	aa.lock.Lock()
	defer aa.lock.Unlock()
	// 读取 dir 文件夹中 所有的 *.html 文件
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".html") {
			return nil
		}
		// 读取文件内容
		txt, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		tpl := &Tpl{}
		tpl.Key = path[len(dir)+1:]
		tpl.Txt = string(txt)
		tpl.Tpl, tpl.Err = template.New(tpl.Key).Parse(tpl.Txt)
		if tpl.Err == nil {
			tpl.Tpl.Funcs(aa.FuncMap)
		}
		aa.tpls[tpl.Key] = tpl
		if debug {
			Println("[_preload]: [tplkit]", tpl.Key)
		}
		return nil
	})
}
