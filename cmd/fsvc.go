package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"k8skit/app/zdb"
	"strings"
	"time"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
	"github.com/suisrc/zgg/z/ze/sqlx"
	"go.yaml.in/yaml/v3"
)

// go run main.go sync

type FixSvcConfig struct {
	User      string
	ConfxKeys z.HM
}

func init() {
	z.CMD["fsvc"] = (&FixSvcCmd{}).fixsvc
	flag.StringVar(&C.CmdFixSvc.User, "fixuser", "fixuser", "k8s fix user")
}

type FixSvcCmd struct {
	zcks *zdb.Zck8sRepo
	conf *zdb.ZconfRepo
}

func (aa *FixSvcCmd) fixsvc() {
	dsx, _ := InitConfig(false)
	if dsx == nil {
		return
	}
	aa.zcks = sqlx.NewRepox[zdb.Zck8sRepo](dsx, nil)
	aa.conf = sqlx.NewRepox[zdb.ZconfRepo](dsx, nil)
	// 删除原有数据
	// aa.zcks.DeleteBy(nil, fmt.Sprintf("version=%d", C.CmdSync.Version))
	// aa.conf.DeleteBy(nil, fmt.Sprintf("version=%d", C.CmdSync.Version))
	// ----------------------------------------------------------
	// 对 yaml 文件进行格式化处理
	aa.fix_() // 调试专用
	//
	z.Println("process, finally")
}

func (aa *FixSvcCmd) fixsvc1(zcks *zdb.Zck8sDO) error {
	if zcks.Kind.String != "Deployment" {
		return fmt.Errorf("kind is %s: %d", zcks.Kind.String, zcks.ID)
	}
	if zcks.Name.String == "" {
		return fmt.Errorf("name is empty: %d", zcks.ID)
	}
	zcks.Name.String = strings.TrimSuffix(zcks.Name.String, "-app")
	name := zcks.Name.String
	data := map[string]any{}
	if err := json.Unmarshal([]byte(zcks.Data.String), &data); err != nil {
		return err
	}
	items := []any{}
	if err := json.Unmarshal([]byte(zcks.Json.String), &items); err != nil {
		return err
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		return fmt.Errorf("item[0] is not map: %d", zcks.ID)
	}
	zcks.Ns2 = zcks.Namespace
	// oldName := zc.MapDef(item, "metadata.name", name)
	// if C.CmdFixSvc.ProdVer > 0 {
	// 	if pzcks, _ := aa.zcks.GetBy(nil, nil, nil, "name=? AND version=?", oldName, C.CmdFixSvc.ProdVer); pzcks.ID > 0 {
	// 		zcks.Ns2 = pzcks.Namespace
	// 	} else if pzcks, _ := aa.zcks.GetBy(nil, nil, nil, "name=? AND version=?", name, C.CmdFixSvc.ProdVer); pzcks.ID > 0 {
	// 		zcks.Ns2 = pzcks.Namespace
	// 	}
	// }
	// 删除 metadata.namespace
	// zc.MapSet(item, "metadata.namespace", nil)
	zc.MapSet(item, "metadata.annotations", nil)
	zc.MapSet(item, "metadata.labels", nil)
	zc.MapSet(item, "spec.progressDeadlineSeconds", nil)
	zc.MapSet(item, "spec.strategy", nil)
	zc.MapSet(item, "spec.template.spec.containers.0.terminationMessagePath", nil)
	zc.MapSet(item, "spec.template.spec.containers.0.terminationMessagePolicy", nil)
	zc.MapSet(item, "spec.template.spec.dnsPolicy", nil)
	zc.MapSet(item, "spec.template.spec.schedulerName", nil)
	zc.MapSet(item, "spec.template.spec.securityContext", nil)
	zc.MapSet(item, "spec.template.spec.terminationGracePeriodSeconds", nil)
	zc.MapSet(item, "spec.template.spec.containers.[0].envFrom", nil)
	zc.MapSet(item, "spec.template.spec.containers.[0].resources", nil)
	zc.MapSet(item, "spec.template.metadata.annotations.redeploy-timestamp", nil)

	zc.MapSet(item, "metadata.name", name)
	zc.MapSet(item, "spec.selector.matchLabels.app", name)
	zc.MapSet(item, "spec.template.metadata.labels.app", name)

	if eok := zc.MapGet(item, "spec.template.metadata.labels.ksidecar/inject"); eok == nil {
		zc.MapSet(item, "spec.template.metadata.labels.ksidecar/inject", "enable")
	}
	zc.MapNew(item, "spec.template.metadata.annotations.[ksidecar/db.config]", ".env")
	// zc.MapSet(item, "spec.template.spec.containers.[0].env.[.name=EXT_CFG_HOST].value", "1234567890")
	// 修复镜像地址
	image := zc.MapDef(item, "spec.template.spec.containers.[0].image", "")
	if strings.HasPrefix(image, imagekv[0]) {
		image = imagekv[1] + image[len(imagekv[0]):]
		zc.MapSet(item, "spec.template.spec.containers.[0].image", image)
	}
	// 修复 imagePullSecrets, docker-registry -> local-registry
	zc.MapSet(item, "spec.template.spec.imagePullSecrets.[.name=docker-registry].name", "local-registry")
	zc.MapSet(item, "spec.template.spec.containers.[0].name", "app")
	// 修复 ksidecar/configmap
	kiv := zc.MapDef(item, "spec.template.metadata.annotations.ksidecar/configmap", "")
	if kiv != "" {
		// 修正 kiv 内容
		kin := FixKsidecarConfigmap(kiv)
		if len(kin) > 0 {
			zc.MapSet(item, "spec.template.metadata.annotations.ksidecar/configmap", kin)
		}
	}
	// 修复 service， 如果存在 -svc的service, 需要补充一个不带-svc的service，并标记 service 过期
	var svc map[string]any
	svcs := []any{}
	for _, item := range items {
		if item, ok := item.(map[string]any); !ok {
		} else if kind, ok := item["kind"].(string); !ok {
		} else if kind != "Service" {
		} else if knam := zc.MapDef(item, "metadata.name", ""); name == "" {
		} else if svc == nil && knam == name {
			svc = item
			// 修改 selector 配置
			zc.MapSet(item, "spec.selector", map[string]any{"app": name})
		} else {
			// 修改 selector 配置
			zc.MapSet(item, "spec.selector", map[string]any{"app": name})
			// 标记 anno 为不推荐使用
			zc.MapSet(item, "metadata.annotations", map[string]any{"suggestions": "deprecated.old"})
			zc.MapSet(item, "metadata.labels", nil)
			if port := zc.MapInt(item, "spec.ports.[.name=http].port", 0); port == 12001 {
				// authx -> 12001 -> 12006
				zc.MapSet(item, "spec.ports.[.name=http]", map[string]any{
					"name":       "http",
					"port":       12006,
					"protocol":   "TCP",
					"targetPort": 12006})
			}
			svcs = append(svcs, item)
		}
	}
	if svc == nil {
		// 增加一个 svc
		svc = map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name": name,
			},
			"spec": map[string]any{
				"ports": []any{
					map[string]any{
						"name":       "http1",
						"port":       80,
						"protocol":   "TCP",
						"targetPort": 80,
					},
					map[string]any{
						"name":       "http",
						"port":       12006,
						"protocol":   "TCP",
						"targetPort": 12006,
					},
				},
				"selector": map[string]any{
					"app": name,
				},
			},
		}
		// svcs = append([]any{svc}, svcs...)
		svcs = append(svcs, svc)
	}

	// 处理配置
	if configmap, ok := data["configmap"].(map[string]any); ok {
		z.Println("fixsvc, configmap... ", len(configmap))
		err := aa.conf.Dsc.WithTx(nil, func(dsc sqlx.Dsc) error {
			for kk, vv := range configmap {
				// 处理变量替换
				vv = FixValueConfigMap(kk, vv)
				// ----------------------------------------------------
				// 将 vv 存入数据库中
				conf, _ := aa.conf.GetBy(dsc, nil, nil, "app=? AND kind='env' AND code=? AND deleted=0", name, kk)
				if conf.Version.Valid {
					conf.Version.Int64 += 1
				} else {
					conf.Version.Valid = true
				}
				conf.Name = sqlx.NewString(zcks.Ns2.String + "/" + name)
				conf.Data = sqlx.NewString(vv.(string))
				conf.Kind = sqlx.NewString("env")
				// 更新或插入数据
				var err error
				if conf.ID > 0 {
					z.Println("fixsvc, configmap... update: ", name, kk)
					conf.Updated = sqlx.NewTime(time.Now())
					conf.Updater = sqlx.NewString(C.CmdFixSvc.User)
					err = aa.conf.UpdateByInf(dsc, conf, "Name", "Kind", "Data", "VBD.Updater", "VBD.Updated", "VBD.Version")
				} else {
					z.Println("fixsvc, configmap... insert: ", name, kk)
					conf.App = sqlx.NewString(name)
					conf.Ver = sqlx.NewString("0.0.0")
					conf.Code = sqlx.NewString(kk)
					conf.Disable.Valid = true
					conf.Deleted.Valid = true
					conf.Created = sqlx.NewTime(time.Now())
					conf.Creater = sqlx.NewString(C.CmdFixSvc.User)
					err = aa.conf.Insert(dsc, conf)
				}
				if err != nil {
					return fmt.Errorf("fixsvc, error: %s, name: %s", err.Error(), name)
				}
			}
			return nil
		})
		if err != nil {
			z.Println(err.Error())
			return err
		}
	}
	// 处理配置文件
	// curl http://confx.dev1.sims-cn.com/ac/v1/content?token=123&version=v2.0.0&name=end-iam-kin-app
	// envConfig := k8sc.MapDef(item, "spec.template.spec.containers.0.env.[.name=EXT_CFG_HOST].value", "")
	// if envConfig != "" {
	// 	zc.MapSet(item, "spec.template.spec.containers.0.env.[.name=EXT_CFG_HOST]", nil)
	// 	z.Println("fixsvc, envConfig... ", oldName, envConfig)
	// 	addr := "http://confx.dev1.sims-cn.com/ac/v1/content"
	// 	token, _ := C.CmdFixSvc.ConfxKeys[oldName]
	// 	if token == "" {
	// 		return fmt.Errorf("fixsvc, error: confx token not found: %s", oldName)
	// 	}
	// 	// 远程获取配置内容
	// 	resp, err := http.Get(fmt.Sprintf("%s?token=%s&version=v2.0.0&name=%s", addr, token, oldName))
	// 	if err != nil {
	// 		return err
	// 	}
	// 	body, _ := io.ReadAll(resp.Body)
	// 	resp.Body.Close()
	// 	// 处理变量替换
	// 	vv := FixValueConfigFile(string(body))
	// 	// ----------------------------------------------------
	// 	zc.MapSet(item, "spec.template.metadata.annotations.[ksidecar/db.config]", ".yaml")
	// 	// 将 vv 存入数据库中
	// 	kk := "/www/application.yaml"
	// 	conf, _ := aa.conf.GetBy(nil, nil, nil, "app=? AND kind='yaml' AND code=? AND deleted=0", name, kk)
	// 	if conf.Version.Valid {
	// 		conf.Version.Int64 += 1
	// 	} else {
	// 		conf.Version.Valid = true
	// 	}
	// 	conf.Name = sqlx.NewString(zcks.Ns2.String + "/" + name)
	// 	conf.Data = sqlx.NewString(vv)
	// 	conf.Kind = sqlx.NewString("yaml")
	// 	// 更新或插入数据
	// 	if conf.ID > 0 {
	// 		z.Println("fixsvc, configfile... update: ", name, kk)
	// 		conf.Updated = sqlx.NewTime(time.Now())
	// 		conf.Updater = sqlx.NewString(C.CmdFixSvc.User)
	// 		err = aa.conf.UpdateByInf(nil, conf, "Name", "Kind", "Data", "VBD.Updater", "VBD.Updated", "VBD.Version")
	// 	} else {
	// 		z.Println("fixsvc, configfile... insert: ", name, kk)
	// 		conf.App = sqlx.NewString(name)
	// 		conf.Ver = sqlx.NewString("0.0.0")
	// 		conf.Code = sqlx.NewString(kk)
	// 		conf.Disable.Valid = true
	// 		conf.Deleted.Valid = true
	// 		conf.Created = sqlx.NewTime(time.Now())
	// 		conf.Creater = sqlx.NewString(C.CmdFixSvc.User)
	// 		err = aa.conf.Insert(nil, conf)
	// 	}
	// 	if err != nil {
	// 		return fmt.Errorf("fixsvc, error: %s, name: %s", err.Error(), name)
	// 	}
	// }
	// --------------------------------------------------------------------------------------------------------------------------
	yaml_txt, err := yaml.Marshal(item)
	if err != nil {
		return err
	}
	yaml_str := string(yaml_txt)
	for _, svc := range svcs {
		if yaml_svc, err := yaml.Marshal(svc); err == nil {
			yaml_str = string(yaml_svc) + "\n---\n" + yaml_str
		}
	}

	zcks.Yaml2 = sqlx.NewString(string(yaml_str))
	json_txt, err := json.MarshalIndent(append([]any{item}, svcs...), "", "  ")
	if err != nil {
		return err
	}
	zcks.Json2 = sqlx.NewString(string(json_txt))
	zcks.Updater = sqlx.NewString(C.CmdFixSvc.User)
	zcks.Updated = sqlx.NewTime(time.Now())
	return aa.zcks.UpdateByInc(nil, zcks, "ns2", "yaml2", "json2", "updater", "updated")
}

func (aa *FixSvcCmd) fixing1(zcks *zdb.Zck8sDO) error {
	if zcks.Kind.String != "Ingress" {
		return fmt.Errorf("kind is %s: %d", zcks.Kind.String, zcks.ID)
	}
	if zcks.Name.String == "" {
		return fmt.Errorf("name is empty: %d", zcks.ID)
	}
	nam0 := zcks.Name.String
	nam0 = strings.TrimSuffix(nam0, "-irs")
	nam0 = strings.TrimPrefix(nam0, "ing-")
	nam1 := "end-" + nam0
	// ingress 中只会有一个 ingress
	items := []any{}
	if err := json.Unmarshal([]byte(zcks.Json.String), &items); err != nil {
		return err
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		return fmt.Errorf("item[0] is not map: %d", zcks.ID)
	}
	zcks.Ns2 = zcks.Namespace
	ings := []any{}
	ingm := map[string]any{}
	// 删除 metadata.namespace
	// zc.MapSet(item, "metadata.namespace", nil)
	zc.MapSet(item, "metadata.name", nam1)
	zc.MapSet(item, "metadata.annotations.[cert-manager.io/cluster-issuer]", nil)
	zc.MapSet(item, "metadata.annotations.[nginx.ingress.kubernetes.io/server-snippet]", nil)
	if txt := zc.MapDef(item, "metadata.annotations.[nginx.ingress.kubernetes.io/configuration-snippet]", ""); txt != "" {
		z.Println("[_warning]: configuration-snippet disabled: ", zcks.ID, zcks.Namespace.String, zcks.Name.String)
		txt = zc.TrimYamlString(txt)
		zc.MapSet(item, "metadata.annotations.[nginx.ingress.kubernetes.io/configuration-snippet]", txt)
	}

	rules := zc.MapKeyVal(item, "spec.rules.*.http.paths.*.backend.service[.name=^fnt].name")
	if len(rules) > 0 {
		tls := zc.MapGet(item, "spec.tls")
		if tts, _ := tls.([]any); tts != nil {
			for i, t := range tts {
				switch t := t.(type) {
				case map[string]any:
					tsn, _ := t["secretName"]
					if tsn, _ := tsn.(string); tsn != "" {
						if idx := strings.IndexByte(tsn, '.'); idx > 0 {
							tsn = "tls-" + tsn[idx+1:]
							tts[i].(map[string]any)["secretName"] = tsn
						}
					}
				}
			}
		}
		ns := zcks.Namespace.String
		version := zcks.Version.Int64
		for i := len(rules) - 1; i >= 0; i-- {
			kv := rules[i]
			kk := kv.K
			v0 := kv.V.(string)
			name := strings.TrimSuffix(v0, "-svc")
			// if strings.Contains(name, "-iam-") {
			// 	nam1 := "end-iam-" + nam0
			// 	zc.MapSet(item, "metadata.name", nam1)
			// }
			if ing, ok := ingm[v0].(map[string]any); ok {
				// 补充一个 backend 即可
				key1 := strings.TrimSuffix(kk, ".backend.service.name")
				val1 := zc.MapSet(item, key1, nil).(map[string]any)
				pre := kk
				if idx := strings.Index(pre, ".http.paths."); idx > 0 {
					pre = pre[:idx]
				}
				zc.MapSet(val1, "backed.service.name", name)
				key2 := key1
				if idx := strings.LastIndexByte(key2, '.'); idx > 0 {
					key2 = key2[:idx] + ".-0"
				}
				zc.MapSet(ing, key2, val1)
				// z.Println("[_backend] val:", key2, zc.ToStr2(ing))
				continue
			}
			annos := map[string]any{}
			slike := `%"name": "` + v0 + `",%`
			if svc, err := aa.zcks.GetBy(nil, nil, nil, "kind in (?,?) and namespace=? and `json` like ? and version =?", "Deployment", "Deployment", ns, slike, version); err != nil {
				z.Println("[_ingress]: error ================================== ", ns, v0, version, err.Error())
				continue
			} else {
				svc1 := []map[string]any{}
				json.Unmarshal([]byte(svc.Json.String), &svc1)
				image := zc.MapDef(svc1[0], "spec.template.spec.containers.[0].image", "")
				image = strings.TrimSuffix(image, "-cdn")
				if strings.HasPrefix(image, imagekv[0]) {
					image = imagekv[1] + image[len(imagekv[0]):]
				}
				annos["frontend/service"] = "frontend:http/"
				annos["frontend/db.fronta"] = name
				annos["frontend/db.frontv.image"] = image
				annos["frontend/db.frontv.imagepath"] = "/www/data"
			}
			key1 := strings.TrimSuffix(kk, ".backend.service.name")
			// z.Println("[_backend] key:", key1)
			val1 := zc.MapSet(item, key1, nil).(map[string]any)
			pre := kk
			if idx := strings.Index(pre, ".http.paths."); idx > 0 {
				pre = pre[:idx]
			}
			zc.MapSet(val1, "backed.service.name", name)
			ing := map[string]any{
				"apiVersion": "networking.k8s.io/v1",
				"kind":       "Ingress",
				"metadata": map[string]any{
					"name":      name,
					"namespace": zcks.Namespace.String,
					"labels": map[string]any{
						"frontend/inject": "enable",
					},
					"annotations": annos,
				},
				"spec": map[string]any{
					"rules": []any{
						map[string]any{
							"host": zc.MapGet(item, pre+".host"),
							"http": map[string]any{
								"paths": []any{val1},
							},
						},
					},
					"tls": tls,
				},
			}
			ings = append(ings, ing)
			ingm[v0] = ing
		}
	}
	yaml_str := ""
	for _, ing := range ings {
		if yaml_ing, err := yaml.Marshal(ing); err == nil {
			if len(yaml_str) > 0 {
				yaml_str = string(yaml_ing) + "\n---\n" + yaml_str
			} else {
				yaml_str = string(yaml_ing)
			}
		}
	}
	// 判断 rules 中的 paths 是否有 值
	if rules = zc.MapKeyVal(item, "spec.rules.*.http.paths.0"); len(rules) > 0 {
		if yaml_ing, err := yaml.Marshal(item); err == nil {
			yaml_str = yaml_str + "\n---\n" + string(yaml_ing)
		}
		ings = append([]any{item}, ings...)
	}
	zcks.Yaml2 = sqlx.NewString(string(yaml_str))
	json_txt, err := json.MarshalIndent(ings, "", "  ")
	if err != nil {
		return err
	}
	zcks.Json2 = sqlx.NewString(string(json_txt))
	zcks.Updater = sqlx.NewString(C.CmdFixSvc.User)
	zcks.Updated = sqlx.NewTime(time.Now())
	return aa.zcks.UpdateByInc(nil, zcks, "ns2", "yaml2", "json2", "updater", "updated")
}

func (aa *FixSvcCmd) fix_() {
	aa.fixing0()
}

func (aa *FixSvcCmd) fixsvc0() {
	zcks, err := aa.zcks.Get(nil, 4596)
	if err != nil {
		fmt.Println("get zck8s error: ", err.Error())
		return
	}
	if err := aa.fixsvc1(zcks); err != nil {
		fmt.Println("fixsvc error: ", err.Error())
		return
	}
}

func (aa *FixSvcCmd) fixing0() {

	if zcks, err := aa.zcks.Get(nil, 4771); err != nil {
		fmt.Println("get zck8s error: ", err.Error())
		return
	} else if err := aa.fixing1(zcks); err != nil {
		fmt.Println("fixing error: ", err.Error())
		return
	}
	// if zcks, err := aa.zcks.SelectBy(nil, nil, "kind=? AND version=? AND deleted=0", "Ingress", C.CmdSync.Version); err != nil {
	// 	fmt.Println("get zck8s error: ", err.Error())
	// 	return
	// } else {
	// 	for _, z1 := range zcks {
	// 		z.Println("[_fixing_]: ", z1.ID, z1.Namespace.String, z1.Name.String)
	// 		if err := aa.fixing1(&z1); err != nil {
	// 			fmt.Println("fixing error: ", err.Error())
	// 			return
	// 		}
	// 	}
	// }
}

func FixKsidecarConfigmap(kiv string) string {
	kin := strings.Builder{}
	for kk := range strings.SplitSeq(kiv, ",") {
		kk := strings.TrimSpace(kk)
		// ----------------------------------------------------
		if _, after, ok := strings.Cut(kk, "prod-kwdog#"); ok {
			kin.WriteRune(',')
			kin.WriteString("default/kwdog#")
			kin.WriteString(after)
		}
		// ----------------------------------------------------
	}
	if kin.Len() > 0 {
		return kin.String()[1:]
	}
	return ""
}

func FixValueConfigMap(key string, val any) any {
	vv := val.(string)
	switch key {
	case "SKY_DATABASE_HOST":
		vv = "mysqlx.base.svc"
	case "SKY_LOGGER_SYSLOGADDR":
		vv = "klog.default.svc:5141"
	case "SKY_REDIS_ADDRESS":
		vv = "redis.base.svc:6379"
	case "SKY_MONGODB_HOST":
		vv = "mongox.base.svc"
	case "SKY_NATS_ADDRESS":
		vv = "nats://natsx.base.svc:4222"
	case "SKY_SERVE_AUTHXSERVER":
		vv = "http://end-iam-kin.rs-iam.svc/authx"
	case "SKY_SERVE_AUTHZSERVER":
		vv = "http://end-iam-kin.rs-iam.svc/authx"
	case "LOGGING_SYSLOG_HOST":
		vv = "klog.default.svc"
	case "SPRING_DATA_MONGODB_HOST":
		vv = "mongox.base.svc"
	case "SPRING_REDIS_HOST", "REDIS_HOST":
		vv = "redis.base.svc"
	case "SPRING_SERVICE_IAMPAS_URL":
		vv = "http://end-fmes-pas.rs-iam.svc"
	case "SPRING_SERVICE_TDUCK_URL":
		vv = "http://end-fmes-tduck.rs-iam.svc"
	case "SPRING_SERVICE_PLATFORM_URL":
		vv = "http://end-iam-pas.rs-iam.svc"
	}
	// 内容识别并进行替换
	if strings.HasPrefix(vv, "http://end-") && strings.Contains(vv, "-svc.") {
		vv = strings.Replace(vv, "-svc.", ".", 1)
	}
	switch vv {
	case "http://end-tas.rs-iam.svc":
		vv = "http://end-fmes-tas.rs-iam.svc"
	case "http://end-pas.rs-iam.svc":
		vv = "http://end-fmes-pas.rs-iam.svc"
	default:
		if strings.Contains(vv, "pc-uf6t3o4p8cs85fg8e.rwlb.rds.aliyuncs.com") {
			vv = strings.Replace(vv, "pc-uf6t3o4p8cs85fg8e.rwlb.rds.aliyuncs.com", "mysqlx.base.svc", 1)
		} else if strings.Contains(vv, "nats-svc:4222") {
			vv = strings.Replace(vv, "nats-svc:4222", "natsx.base.svc:4222", 1)
		} else if strings.Contains(vv, "nats:4222") {
			vv = strings.Replace(vv, "nats:4222", "natsx.base.svc:4222", 1)
		}
	}
	return vv
}

func FixValueConfigFile(vv string) string {
	vv = strings.ReplaceAll(vv, "mysql-svc", "mysqlx.base.svc")
	vv = strings.ReplaceAll(vv, "mongo-svc", "mongox.base.svc")
	vv = strings.ReplaceAll(vv, "nats-svc:4222", "natsx.base.svc:4222")
	vv = strings.ReplaceAll(vv, "redis-svc", "redis.base.svc")
	vv = strings.ReplaceAll(vv, "http://end-iam-kin-svc/authx", "http://end-iam-kin.rs-iam.svc/authx")
	vv = strings.ReplaceAll(vv, "http://end-iam-kin-svc/authz", "http://end-iam-kin.rs-iam.svc/authz")
	vv = strings.ReplaceAll(vv, "log-svc:5140", "klog.default.svc:5141")
	vv = strings.ReplaceAll(vv, "log-svc", "klog.default.svc")
	vv = strings.ReplaceAll(vv, "http://end-fmes-pas-svc", "http://end-fmes-pas.rs-iam.svc")
	return vv
}

var imagekv [2]string = [2]string{"registry-vpc.cn-shanghai.aliyuncs.com/fmes/", "plus/"}
