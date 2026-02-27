package front3

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/ze/sqlx"
	"go.yaml.in/yaml/v2"
	netv1 "k8s.io/api/networking/v1"
)

func (aa *Serve) mutateLogIngress(old *netv1.Ingress, ing *netv1.Ingress, raw []byte) { // 记录网关数据
	if !C.Front3.LogIngress {
		return // 不记录
	}
	if old != nil && (ing == nil || ing.Name != old.Name) {
		// 删除所有的旧版本
		if ados, err := aa.IngRepo.GetBySpaceAndNames(old.Namespace, old.Name); err != nil && err != sql.ErrNoRows {
			z.Println("[_mutate_]:", "get ingress form database error,", err.Error())
			return // 数据库异常
		} else if len(ados) > 0 {
			for _, ado := range ados {
				ado.Deleted = true
				aa.IngRepo.DeleteOne(&ado)
			}
		}
	}
	if ing == nil {
		return // 删除时该字段不存在
	}
	z.Println("[_mutate_]: log ingress to database,", ing.Namespace, "|", ing.Name)
	ado, err := aa.IngRepo.GetBySpaceAndName(ing.Namespace, ing.Name)
	if err != nil && err != sql.ErrNoRows {
		z.Println("[_mutate_]:", "get ingress form database error,", err.Error())
		return // 数据库异常
	}
	clzz := ""
	if ing.Spec.IngressClassName != nil {
		clzz = *ing.Spec.IngressClassName
	}
	ado.Ns = sqlx.NewString(ing.Namespace)
	ado.Name = sqlx.NewString(ing.Name)
	ado.Clzz = sqlx.NewString(clzz)
	ado.Host = sqlx.NewString(getIngressHosts(ing)[0])
	// ado.Template = sql.NullString{String: string(raw), Valid: true}
	// 使用 yaml 格式存储
	{
		obj := map[string]any{}
		if err := json.Unmarshal(raw, &obj); err == nil {
			delete(obj, "status")
			if mate, ok := obj["metadata"].(map[string]any); ok {
				// 补充扩展信息
				uid_, _ := mate["uid"].(string)
				ver_, _ := mate["resourceVersion"].(string)
				ado.MetaUid = sqlx.NewString(uid_)
				ado.MetaVer = sqlx.NewString(ver_)
				// 删除扩展信息
				delete(mate, "uid")
				delete(mate, "resourceVersion")
				delete(mate, "generation")
				delete(mate, "creationTimestamp")
				delete(mate, "managedFields")
				if anno, ok := mate["annotations"].(map[string]any); ok {
					delete(anno, "kubectl.kubernetes.io/last-applied-configuration")
				}
			}
		}
		template, _ := yaml.Marshal(obj)
		ado.Template = sql.NullString{String: string(template), Valid: true}
	}
	ado.Disable = false
	ado.Deleted = false
	if ado.ID > 0 {
		ado.Updated = sql.NullTime{Time: time.Now(), Valid: true}
		ado.Updater = sql.NullString{String: z.AppName, Valid: true}
		aa.IngRepo.UpdateOne(ado)
	} else {
		ado.ID = 0
		ado.Created = sql.NullTime{Time: time.Now(), Valid: true}
		ado.Creater = sql.NullString{String: z.AppName, Valid: true}
		// ado.Updated = ado.Created
		// ado.Updater = ado.Creater
		aa.IngRepo.InsertOne(ado)
	}
}
