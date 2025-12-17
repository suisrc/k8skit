package sidecar

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// PatchOperation JsonPatch struct http://jsonpatch.com/
type PatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

func baseAdmissionReview() *admissionv1.AdmissionReview {
	gvk := admissionv1.SchemeGroupVersion.WithKind("AdmissionReview")
	return &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
	}
}

func admissionResponse(httpErrorCode int, message string) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Result: &metav1.Status{
			Code:    int32(httpErrorCode),
			Message: message,
		},
	}
}

func errorAdmissionReview(httpErrorCode int, message string) *admissionv1.AdmissionReview {
	r := baseAdmissionReview()
	r.Response = admissionResponse(httpErrorCode, message)
	return r
}

func writeAdmissionReview(r *admissionv1.AdmissionReview, res http.ResponseWriter) {
	resp, err := json.Marshal(r)
	if err != nil {
		klog.Errorf("Error marshalling decision: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = res.Write(resp)
	if err != nil {
		klog.Errorf("Error writing response: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func writeErrorAdmissionReview(status int, message string, res http.ResponseWriter) {
	admResp := errorAdmissionReview(status, message)
	writeAdmissionReview(admResp, res)
}

func writeDeniedAdmissionResponse(ar *admissionv1.AdmissionReview, message string, res http.ResponseWriter) {
	ar.Response = admissionResponse(http.StatusForbidden, message)
	ar.Response.UID = ar.Request.UID
	writeAdmissionReview(ar, res)
}

func writeAllowedAdmissionReview(ar *admissionv1.AdmissionReview, patch []byte, res http.ResponseWriter) {
	ar.Response = admissionResponse(http.StatusOK, "")
	ar.Response.Allowed = true
	ar.Response.UID = ar.Request.UID
	if patch != nil {
		pt := admissionv1.PatchTypeJSONPatch
		ar.Response.Patch = patch
		ar.Response.PatchType = &pt
	}
	writeAdmissionReview(ar, res)
}

func unmarshalPod(rawObject []byte) (corev1.Pod, error) {
	var pod corev1.Pod
	err := json.Unmarshal(rawObject, &pod)
	return pod, errors.Wrapf(err, "error unmarshalling object")
}
