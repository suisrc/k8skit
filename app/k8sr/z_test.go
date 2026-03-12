package k8s_test

import (
	"k8skit/app/k8sc"
	"os"
	"strings"
	"testing"

	"github.com/suisrc/zgg/z"
	"gopkg.in/yaml.v3"
)

// go test -v app/k8sr/z_test.go -run Test_yaml

type LiteralString string

func (s LiteralString) MarshalYAML() (any, error) {
	str := string(s)
	// 仅对多行字符串使用|-样式
	if strings.Contains(str, "\n") {
		println(" ---------------------------------- ")
		println(str)
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: str,
			Style: yaml.LiteralStyle,
		}, nil
	}
	// 单行字符串使用普通样式
	return str, nil
}

func Test_yaml(t *testing.T) {
	t.Log("hello world!")

	bts, _ := os.ReadFile("../../_out/_temp.yaml")
	// t.Log(string(bts))
	tmp := map[string]any{}
	if err := yaml.Unmarshal(bts, &tmp); err != nil {
		t.Fatal(err)
	}
	if data, ok := tmp["data"]; !ok {
	} else if data, ok := data.(map[string]any); !ok {
	} else {
		for k, v := range data {
			v := v.(string)
			// data[k] = LiteralString(v)
			if strings.ContainsRune(v, '\n') {
				data[k] = k8sc.TrimYamlString(v)
			}
		}
	}
	bts, _ = yaml.Marshal(tmp)
	os.WriteFile("../../_out/_temp1.yaml", bts, 0644)
	// t.Log(tmp)
}

// go test -v app/k8sr/z_test.go -run Test_mapdef

func Test_mapdef(t *testing.T) {
	dmap := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": "d",
				"d": 123,
				"e": "456",
				"f": 456.789,
				"g": uint16(0),
				"i": "N",
				"a": map[string]any{
					"j": "123",
				},
				"b": map[string]any{
					"j": "321",
				},
			},
		},
	}

	t.Log("=================== ", k8sc.MapAny(dmap, "a.b.e", false))
	t.Log("=================== ", k8sc.MapAny(dmap, "a.b.f", false))
	t.Log("=================== ", k8sc.MapAny(dmap, "a.b.g", true))
	t.Log("=================== ", k8sc.MapAny(dmap, "a.b.h", false))
	t.Log("=================== ", k8sc.MapAny(dmap, "a.b.i", true))
	k8sc.MapVaz(dmap, "a.b.x.y.-0.v", "123")
	t.Log("=================== ", k8sc.MapAny(dmap, "a.b.x.y.0.v", 0))
	k8sc.MapVaz(dmap, "a.b.x.y.0.v", "456")
	t.Log("=================== ", k8sc.MapAny(dmap, "a.b.x.y.0.v", 0))
	// k8sc.MapVaz(dmap, "a.b.x.y.0", nil)
	k8sc.MapVaz(dmap, "a.b.x.y.-0.-0.z", "123")
	t.Log("=================== ", k8sc.MapAny(dmap, "a.b.x.y.0.0.z", 0))
	k8sc.MapVaz(dmap, "a.b.x.y.-0.-0.z", "123")
	k8sc.MapVaz(dmap, "a.b.x.y.1.-0.z", "789")
	k8sc.MapVaz(dmap, "a.b.x.y.1.-0.z", "567")
	k8sc.MapVaz(dmap, "a.b.x.y.-1.-0.z", "234")
	t.Log(z.ToStr2(dmap))
	t.Log("=================== ", k8sc.MapAny(dmap, "a.b.x.y.1.[.z=^*.6].z", 0))
	t.Log("=================== ", k8sc.MapKey(dmap, "a.b.[.j=^*.2].j"))
}
