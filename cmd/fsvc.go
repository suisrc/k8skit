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
	ProdVer   int
}

func init() {
	z.CMD["fsvc"] = (&FixSvcCmd{}).fixsvc
	flag.StringVar(&C.CmdFixSvc.User, "fixuser", "fixuser", "k8s fix user")
	flag.IntVar(&C.CmdFixSvc.ProdVer, "fixprodver", 13, "k8s fix prod version")
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
	aa.fixsvc0() // 调试专用
	//
	z.Println("fixsvc, finally")
}

func (aa *FixSvcCmd) fixsvc1(zcks *zdb.Zck8sDO) error {
	if zcks.Kind.String != "Deployment" {
		return fmt.Errorf("kind is Deployment: %d", zcks.ID)
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
	oldName := zc.MapDef(item, "metadata.name", name)
	zcks.Ns2 = zcks.Namespace
	if C.CmdFixSvc.ProdVer > 0 {
		if pzcks, _ := aa.zcks.GetBy(nil, nil, nil, "name=? AND version=?", oldName, C.CmdFixSvc.ProdVer); pzcks.ID > 0 {
			zcks.Ns2 = pzcks.Namespace
		} else if pzcks, _ := aa.zcks.GetBy(nil, nil, nil, "name=? AND version=?", name, C.CmdFixSvc.ProdVer); pzcks.ID > 0 {
			zcks.Ns2 = pzcks.Namespace
		}
	}
	// 删除 metadata.namespace
	// zc.MapVal(item, "metadata.namespace", nil)
	zc.MapVal(item, "metadata.annotations", nil)
	zc.MapVal(item, "metadata.labels", nil)
	zc.MapVal(item, "spec.progressDeadlineSeconds", nil)
	zc.MapVal(item, "spec.strategy", nil)
	zc.MapVal(item, "spec.template.spec.containers.0.terminationMessagePath", nil)
	zc.MapVal(item, "spec.template.spec.containers.0.terminationMessagePolicy", nil)
	zc.MapVal(item, "spec.template.spec.dnsPolicy", nil)
	zc.MapVal(item, "spec.template.spec.schedulerName", nil)
	zc.MapVal(item, "spec.template.spec.securityContext", nil)
	zc.MapVal(item, "spec.template.spec.terminationGracePeriodSeconds", nil)
	zc.MapVal(item, "spec.template.spec.containers.[0].envFrom", nil)
	zc.MapVal(item, "spec.template.spec.containers.[0].resources", nil)
	zc.MapVal(item, "spec.template.metadata.annotations.redeploy-timestamp", nil)

	zc.MapVal(item, "metadata.name", name)
	zc.MapVal(item, "spec.selector.matchLabels.app", name)
	zc.MapVal(item, "spec.template.metadata.labels.app", name)
	zc.MapVal(item, "spec.template.metadata.annotations.[ksidecar/db.config]", ".env")
	// zc.MapVal(item, "spec.template.spec.containers.[0].env.[.name=EXT_CFG_HOST].value", "1234567890")
	// 修复镜像地址
	image := zc.MapDef(item, "spec.template.spec.containers.[0].image", "")
	if strings.HasPrefix(image, "registry-vpc.cn-shanghai.aliyuncs.com/fmes/") {
		image = "dcr.dev.sims-cn.com/plus/" + image[len("registry-vpc.cn-shanghai.aliyuncs.com/fmes/"):]
		zc.MapVal(item, "spec.template.spec.containers.[0].image", image)
	}
	// 修复 ksidecar/configmap
	kiv := zc.MapDef(item, "spec.template.metadata.annotations.ksidecar/configmap", "")
	if kiv != "" {
		// 修正 kiv 内容
		kin := FixKsidecarConfigmap(kiv)
		if len(kin) > 0 {
			zc.MapVal(item, "spec.template.metadata.annotations.ksidecar/configmap", kin)
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
			zc.MapVal(item, "spec.selector", map[string]any{"app": name})
		} else {
			// 修改 selector 配置
			zc.MapVal(item, "spec.selector", map[string]any{"app": name})
			// 标记 anno 为不推荐使用
			zc.MapVal(item, "metadata.annotations", map[string]any{"suggestions": "deprecated"})
			zc.MapVal(item, "metadata.labels", nil)
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
	// 	k8sc.MapVal(item, "spec.template.spec.containers.0.env.[.name=EXT_CFG_HOST]", nil)
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
	// 	k8sc.MapVal(item, "spec.template.metadata.annotations.[ksidecar/db.config]", ".yaml")
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
	return aa.zcks.UpdateByInc(nil, zcks, "ns2", "name", "yaml2", "json2", "updater", "updated")
}

func (aa *FixSvcCmd) fixsvc0() {
	zcks, err := aa.zcks.Get(nil, 4406)
	if err != nil {
		fmt.Println("get zck8s error: ", err.Error())
		return
	}
	if err := aa.fixsvc1(zcks); err != nil {
		fmt.Println("fixsvc error: ", err.Error())
		return
	}
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

func FixValueConfigMap(kk string, vv any) any {
	switch kk {
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
	case "NATS_SPRING_SERVER":
		vv = strings.ReplaceAll(vv.(string), "nats-svc", "natsx.base.svc")
	case "SPRING_DATASOURCE_URL":
		vv = strings.ReplaceAll(vv.(string), "mysql-svc", "mysqlx.base.svc")
	case "SPRING_DATA_MONGODB_HOST":
		vv = "mongox.base.svc"
	case "SPRING_REDIS_HOST":
		vv = "redis.base.svc"
	case "SPRING_SERVICE_IAMPAS_URL":
		vv = "http://end-fmes-pas.rs-iam.svc"
	case "SPRING_SERVICE_TDUCK_URL":
		vv = "http://end-fmes-tduck.rs-iam.svc"
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
