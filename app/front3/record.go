package front3

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/suisrc/zgg/z"
	"go.yaml.in/yaml/v2"
	admissionv1 "k8s.io/api/admission/v1"
)

func (aa *Serve) Record(rw http.ResponseWriter, rr *http.Request) {
	if err := checkPostJson(rr); err != nil {
		z.Println(err.Error())
		writeErrorAdmissionReview(http.StatusBadRequest, err.Error(), rw)
		return
	}
	admReview, err := z.ReadBody(rr, &admissionv1.AdmissionReview{})
	if err != nil {
		z.Printf("Could not decode body: %v", err)
		writeErrorAdmissionReview(http.StatusInternalServerError, err.Error(), rw)
		return
	}
	req := admReview.Request

	z.Printf("AdmissionReview for Kind=%v, Namespace=%v Name=%v UID=%v patchOperation=%v UserInfo=%v", //
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)

	if patchOperations, err := aa.recordProcess(req); err != nil {
		message := fmt.Sprintf("request for object '%s' with name '%s' in namespace '%s' denied: %v", //
			req.Kind.String(), req.Name, req.Namespace, err)
		z.Println(message)
		writeDeniedAdmissionResponse(admReview, message, rw)
	} else if /*len(patchOperations) == 0*/ patchOperations == nil {
		writeAllowedAdmissionReview(admReview, nil, rw)
	} else if patchBytes, err := json.Marshal(patchOperations); err != nil {
		message := fmt.Sprintf("request for object '%s' with name '%s' in namespace '%s' denied: %v", //
			req.Kind.String(), req.Name, req.Namespace, err)
		z.Println(message)
		writeDeniedAdmissionResponse(admReview, message, rw)
	} else {
		writeAllowedAdmissionReview(admReview, patchBytes, rw)
	}
}

func (aa *Serve) recordProcess(req *admissionv1.AdmissionRequest) ([]PatchOperation, error) {
	// 处理
	// patchs := []PatchOperation{}
	switch req.Operation {
	case admissionv1.Create: // 创建, 更新或者创建应用
		raw := map[string]any{}
		if err := json.Unmarshal(req.Object.Raw, &raw); err != nil {
			return nil, errors.New("error unmarshalling new object, " + err.Error())
		}
		aa.recordSave(nil, raw)
	case admissionv1.Update: // 更新, 更新应用版本信息
		old := map[string]any{}
		if err := json.Unmarshal(req.OldObject.Raw, &old); err != nil {
			return nil, errors.New("error unmarshalling old object, " + err.Error())
		}
		raw := map[string]any{}
		if err := json.Unmarshal(req.Object.Raw, &raw); err != nil {
			return nil, errors.New("error unmarshalling new object, " + err.Error())
		}
		aa.recordSave(old, raw)
	case admissionv1.Delete: // 删除, 对应服务逻辑删除
		old := map[string]any{}
		if err := json.Unmarshal(req.OldObject.Raw, &old); err != nil {
			return nil, errors.New("error unmarshalling old object, " + err.Error())
		}
		aa.recordSave(old, nil)
	default:
		return nil, fmt.Errorf("unhandled request operations type %s", req.Operation)
	}
	// return patchs, nil
	return nil, nil
}

func (aa *Serve) recordSave(old, raw map[string]any) { // 记录网关数据
	version := 0
	if old != nil {
		// 删除旧版本
		if mateold, ok := old["metadata"].(map[string]any); !ok {
			z.Println("[_record_]:", "get old metadata is not found.")
			return
		} else if namespace, ok := mateold["namespace"].(string); !ok {
			z.Println("[_record_]:", "get old metadata.namespace is not found.")
			return
		} else if name, ok := mateold["name"].(string); !ok {
			z.Println("[_record_]:", "get old metadata.name is not found.")
			return
		} else if ads, err := aa.RecRepo.LstByNamespaceAndNameAndDeleted(namespace, name, false); err != nil && err != sql.ErrNoRows {
			z.Println("[_record_]:", "get object form database error,", err.Error())
			return // 数据库异常
		} else if len(ads) > 0 {
			for _, ado := range ads {
				if version < ado.Version {
					version = ado.Version
				}
				ado.Deleted = true
				ado.Updated = sql.NullTime{Time: time.Now(), Valid: true}
				if raw == nil {
					ado.Updater = sql.NullString{String: "deleted", Valid: true}
				} else {
					ado.Updater = sql.NullString{String: "updated", Valid: true}
				}
				aa.RecRepo.DeleteOne(&ado)
			}
		}
	}
	if raw == nil {
		return // 删除时该字段不存在
	}
	// var ing netv1.Ingress
	rawna := ""
	rawns := ""
	if mate, ok := raw["metadata"].(map[string]any); ok {
		rawna, _ = mate["name"].(string)
		rawns, _ = mate["namespace"].(string)
	} else {
		z.Println("[_record_]:", "get new metadata is not found.")
		return
	}
	z.Println("[_record_]: log record to database,", rawns, "|", rawna)
	ado := &RecordDO{}
	// 基础信息
	ado.Version = version
	ado.Created = sql.NullTime{Time: time.Now(), Valid: true}
	ado.Creater = sql.NullString{String: "zrecord", Valid: true}
	ado.Updated = ado.Created
	ado.Updater = sql.NullString{String: "created", Valid: true}
	ado.Disable = false
	ado.Deleted = false
	// 记录信息
	ado.Namespace = sql.NullString{String: rawns, Valid: true}
	ado.Name = sql.NullString{String: rawna, Valid: true}
	if kind, ok := raw["kind"].(string); ok {
		ado.Kind = sql.NullString{String: kind, Valid: true}
	}
	if apiv, ok := raw["apiVersion"].(string); ok {
		ado.ApiVersion = sql.NullString{String: apiv, Valid: true}
	}
	delete(raw, "status") // 删除状态字段
	if mate, ok := raw["metadata"].(map[string]any); ok {
		// 补充扩展信息
		uid_, _ := mate["uid"].(string)
		ver_, _ := mate["resourceVersion"].(string)
		ado.MetaUid = sql.NullString{String: uid_, Valid: true}
		ado.MetaVer = sql.NullString{String: ver_, Valid: true}
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
	template, _ := yaml.Marshal(raw)
	ado.Template = sql.NullString{String: string(template), Valid: true}
	// 保存记录
	aa.RecRepo.InsertOne(ado)
}
