package z

// 服务汇总

import (
	"fmt"
	"maps"
	"reflect"
	"sync"
)

/**
 * 注册服务, key 必须唯一, 如果 key 为空， 使用 val.(type).Name() 作为 key
 * @param kit 服务容器
 * @param inj 自动注入
 * @param key 服务 key
 * @param val 服务实例
 */
func RegKey[T any](kit SvcKit, inj bool, key string, val T) T {
	if key == "" {
		key = reflect.TypeOf(val).Elem().Name()
	}
	kit.Set(key, val)
	if inj {
		kit.Inj(val) // 自动注入， 可以注入自己
	}
	return val
}

// 注册服务， name = val.(type)
func RegSvc[T any](kit SvcKit, val T) T {
	key := reflect.TypeOf(val).Elem().Name()
	kit.Set(key, val)
	return val
}

// 注册服务， name = val.(type)， 并自动初始化 val 实体
func Inject[T any](kit SvcKit, val T) T {
	key := reflect.TypeOf(val).Elem().Name()
	kit.Set(key, val).Inj(val) // 自动注入， 可以注入自己
	return val
}

// ----------------------------------------------------------------------------
// ----------------------------------------------------------------------------

var _ SvcKit = (*SvcKit0)(nil)

type SvcKit0 struct {
	debug  bool
	server IServer
	svcmap map[string]any
	typmap map[reflect.Type]any
	svclck sync.RWMutex
}

func NewSvcKit(server IServer, debug bool) SvcKit {
	svckit := &SvcKit0{
		debug:  debug,
		server: server,
		svcmap: make(map[string]any),
		typmap: make(map[reflect.Type]any),
	}
	svckit.svcmap["svckit"] = svckit
	svckit.svcmap["server"] = server
	svckit.svcmap["tplkit"] = server.GetTplKit()
	svckit.svcmap["config"] = server.GetConfig()
	svckit.typmap[reflect.TypeOf(svckit)] = svckit
	svckit.typmap[reflect.TypeOf(server)] = server
	return svckit
}

func (aa *SvcKit0) Srv() IServer {
	return aa.server
}

func (aa *SvcKit0) Get(key string) any {
	aa.svclck.RLock()
	defer aa.svclck.RUnlock()
	return aa.svcmap[key]
}

func (aa *SvcKit0) Set(key string, val any) SvcKit {
	aa.svclck.Lock()
	defer aa.svclck.Unlock()
	if val != nil {
		// create or update
		aa.svcmap[key] = val
		aa.typmap[reflect.TypeOf(val)] = val
	} else {
		// delete
		val := aa.svcmap[key]
		if val != nil {
			delete(aa.svcmap, key)
			// delete value by type
			for kk, vv := range aa.typmap {
				if vv == val {
					delete(aa.typmap, kk)
					break
				}
			}
		}
	}
	return aa
}

func (aa *SvcKit0) Map() map[string]any {
	aa.svclck.RLock()
	defer aa.svclck.RUnlock()
	ckv := make(map[string]any)
	maps.Copy(ckv, aa.svcmap)
	return ckv
}

func (aa *SvcKit0) Inj(obj any) SvcKit {
	aa.svclck.RLock()
	defer aa.svclck.RUnlock()
	// 构建注入映射
	tType := reflect.TypeOf(obj).Elem()
	tElem := reflect.ValueOf(obj).Elem()
	for i := 0; i < tType.NumField(); i++ {
		tField := tType.Field(i)
		tagVal := tField.Tag.Get("svckit")
		if tagVal == "" || tagVal == "-" {
			continue // 忽略
		}
		if tagVal == "type" || tagVal == "auto" {
			// 通过 `svckit:'type/auto'` 中的接口匹配注入
			found := false
			for vType, value := range aa.typmap {
				if tField.Type == vType || // 属性是一个接口，判断接口是否可以注入
					tField.Type.Kind() == reflect.Interface && vType.Implements(tField.Type) {
					tElem.Field(i).Set(reflect.ValueOf(value))
					if aa.debug {
						Printf("[_svckit_]: [inject] %s.%s <- %s\n", tType, tField.Name, vType)
					}
					found = true
					break
				}
			}
			if !found {
				errstr := fmt.Sprintf("[_svckit_]: [inject] %s.%s <- %s.(type) error not found", tType, tField.Name, tField.Type)
				if aa.debug {
					Println(errstr)
				} else {
					Fatal(errstr) // 生产环境，注入失败，则 panic
				}
			}
		} else {
			// 通过 `svckit:'[name]'` 中的 [name] 注入
			val := aa.svcmap[tagVal]
			if val == nil {
				errstr := fmt.Sprintf("[_svckit_]: [inject] %s.%s <- %s.[name] error not found", tType, tField.Name, tagVal)
				if aa.debug {
					Println(errstr)
				} else {
					Fatal(errstr) // 生产环境，注入失败，则 panic
				}
				continue
			}
			tElem.Field(i).Set(reflect.ValueOf(val))
			if aa.debug {
				Printf("[_svckit_]: [inject] %s.%s <- %s\n", tType, tField.Name, reflect.TypeOf(val))
			}
		}
	}
	return aa
}
