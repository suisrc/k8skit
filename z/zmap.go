package z

import (
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
)

// --------------------------------------------------------------------------

// ToStr ...
func ToStr(aa any) string {
	if bts, err := json.Marshal(aa); err != nil {
		return ""
	} else {
		return string(bts)
	}
}

// AsMap ...
func AsMap(aa any) map[string]any {
	ref := reflect.ValueOf(aa)
	if ref.Kind() != reflect.Map {
		return nil // panic("obj is not map")
	}
	rss := make(map[string]any)
	for _, key := range ref.MapKeys() {
		rss[key.String()] = ref.MapIndex(key).Interface()
	}
	return rss
}

// --------------------------------------------------------------------------

type Tag struct {
	Tags  []string
	Field reflect.StructField
	Value reflect.Value
}

func ToTag(val any, tag string) ([]*Tag, reflect.Kind) {
	vtype := reflect.TypeOf(val)
	value := reflect.ValueOf(val)
	if vtype.Kind() == reflect.Pointer {
		vtype = vtype.Elem()
		value = value.Elem()
	}
	vkind := vtype.Kind()
	if vkind != reflect.Struct {
		return nil, vkind
	}
	tags := []*Tag{}
	for i := 0; i < vtype.NumField(); i++ {
		field := vtype.Field(i)
		vtags := []string{}
		if tag != "" {
			tagVal := field.Tag.Get(tag)
			if tagVal == "-" {
				continue
			}
			vtags = strings.Split(tagVal, ",")
		}
		if len(vtags) == 0 {
			vtags = []string{strings.ToLower(field.Name)}
		}
		tags = append(tags, &Tag{
			Tags:  vtags,
			Field: field,
			Value: value.Field(i),
		})
	}
	return tags, vkind
}

// ToMap ...
func ToMap(target any, tagName string) map[string]any {
	tags, kind := ToTag(target, tagName)
	if kind == reflect.Map {
		return AsMap(target)
	} else if kind != reflect.Struct {
		return nil // panic("obj is not struct")
	}
	rss := make(map[string]any)
	for _, tag := range tags {
		rss[tag.Tags[0]] = tag.Value.Interface()
	}
	return rss
}

// ByMap ...
func ByMap[T any](target T, source map[string]any, tagName string) (T, error) {
	tags, kind := ToTag(target, tagName)
	if kind == reflect.Map {
		value := reflect.ValueOf(target)
		for kk, vv := range source {
			value.SetMapIndex(reflect.ValueOf(kk), reflect.ValueOf(vv))
		}
		return target, nil
	} else if kind != reflect.Struct {
		return target, errors.New("target type is not map or struct")
	}
	for _, tag := range tags {
		value := source[tag.Tags[0]]
		if value == nil {
			continue
		}
		tag.Value.Set(reflect.ValueOf(value))
	}
	return target, nil
}

// --------------------------------------------------------------------------
// --------------------------------------------------------------------------

// ByTag ... use for url.Values
func ByTag[T any](target T, source map[string][]string, tagName string) (T, error) {
	tags, kind := ToTag(target, tagName)
	if kind != reflect.Struct {
		return target, errors.New("target type is not struct")
	}
	for _, tag := range tags {
		val := source[tag.Tags[0]]
		if val == nil {
			continue
		}
		vvv := ToValByStr(tag.Field.Type, val)
		if vvv != nil {
			tag.Value.Set(reflect.ValueOf(vvv))
		}
	}
	return target, nil
}

// ToValByStr ...
func ToValByStr(typ reflect.Type, val []string) any {
	switch typ.Kind() {
	case reflect.String:
		return val[0]
	case reflect.Bool:
		vvv, _ := strconv.ParseBool(val[0])
		return vvv
	case reflect.Int:
		vvv, _ := strconv.Atoi(val[0])
		return vvv
	case reflect.Int64:
		vvv, _ := strconv.ParseInt(val[0], 10, 64)
		return vvv
	case reflect.Uint:
		vvv, _ := strconv.ParseUint(val[0], 10, 64)
		return uint(vvv)
	case reflect.Uint64:
		vvv, _ := strconv.ParseUint(val[0], 10, 64)
		return vvv
	case reflect.Float64:
		vvv, _ := strconv.ParseFloat(val[0], 64)
		return vvv
	case reflect.Slice:
		// slice, []any...
		switch typ.Elem().Kind() {
		case reflect.String:
			return val[:]
		case reflect.Bool:
			vva := []bool{}
			for _, vv := range val {
				vvb, _ := strconv.ParseBool(vv)
				vva = append(vva, vvb)
			}
			return vva
		case reflect.Int:
			vva := []int{}
			for _, vv := range val {
				vvb, _ := strconv.Atoi(vv)
				vva = append(vva, vvb)
			}
			return vva
		case reflect.Int64:
			vva := []int64{}
			for _, vv := range val {
				vvb, _ := strconv.ParseInt(vv, 10, 64)
				vva = append(vva, vvb)
			}
			return vva
		case reflect.Uint:
			vva := []uint{}
			for _, vv := range val {
				vvb, _ := strconv.ParseUint(vv, 10, 64)
				vva = append(vva, uint(vvb))
			}
			return vva
		case reflect.Uint64:
			vva := []uint64{}
			for _, vv := range val {
				vvb, _ := strconv.ParseUint(vv, 10, 64)
				vva = append(vva, vvb)
			}
			return vva
		case reflect.Float64:
			vva := []float64{}
			for _, vv := range val {
				vvb, _ := strconv.ParseFloat(vv, 64)
				vva = append(vva, vvb)
			}
			return vva
		}
	}
	return nil
}
