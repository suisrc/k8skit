package k8s_test

import (
	"k8skit/app/k8sc"
	"os"
	"strings"
	"testing"

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
				data[k] = k8sc.FormatYamlString(v)
			}
		}
	}
	bts, _ = yaml.Marshal(tmp)
	os.WriteFile("../../_out/_temp1.yaml", bts, 0644)
	// t.Log(tmp)
}
