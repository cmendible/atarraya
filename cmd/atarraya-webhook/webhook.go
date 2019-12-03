package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

var (
	requiredAnnotation  = "atarraya/keyvault"
	statusAnnotationKey = "atarraya/status"
)

const (
	// StatusInjected is the annotation value for /status that indicates an injection was already performed on this pod
	StatusInjected = "injected"
)

type webhookServer struct {
	Server *http.Server
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	var pod corev1.Pod
	if err := json.Unmarshal(ar.Request.Object.Raw, &pod); err != nil {
		glog.Errorf("Could not unmarshal raw object: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	metadata := pod.ObjectMeta

	annotations := metadata.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	status, ok := annotations[statusAnnotationKey]
	if ok && strings.ToLower(status) == StatusInjected {
		glog.Infof("Pod %s/%s annotation %s=%s indicates injection already satisfied, skipping", metadata.Namespace, metadata.Name, statusAnnotationKey, status)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	// determine whether to perform mutation based on annotation for the target resource
	keyvaultName, ok := annotations[requiredAnnotation]
	if !ok {
		// glog.Infof("Pod %s/%s annotation %s is missing, skipping injection", metadata.Namespace, metadata.Name, requestedInjection)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	sidecar := corev1.Container{
		Name:  "az-keyvault-reader-sidecar",
		Image: "cmendibl3/az-keyvault-reader:0.2",
		Env: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  "AZURE_KEYVAULT_NAME",
				Value: keyvaultName,
			},
		},
		ImagePullPolicy: corev1.PullAlways,
	}
	//     resources:
	//       requests:
	//         memory: "8Mi"
	//         cpu: "100m"
	//       limits:
	//         memory: "16Mi"
	//         cpu: "100m"

	var patch []patchOperation

	sidecars := []corev1.Container{sidecar}

	defaulter.Default(&corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: sidecars,
		},
	})

	patch = append(patch, addContainers(pod.Spec.Containers, sidecars, "/spec/containers")...)

	newAnnotations := map[string]string{}
	newAnnotations[statusAnnotationKey] = StatusInjected

	patch = append(patch, updateAnnotations(pod.Annotations, newAnnotations)...)

	patchBytes, err := json.Marshal(patch)

	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func updateAnnotations(target map[string]string, added map[string]string) (patch []patchOperation) {
	for key, value := range added {
		keyEscaped := strings.Replace(key, "/", "~1", -1)

		if target == nil || target[key] == "" {
			target = map[string]string{}
			patch = append(patch, patchOperation{
				Op:    "add",
				Path:  "/metadata/annotations/" + keyEscaped,
				Value: value,
			})
		} else {
			patch = append(patch, patchOperation{
				Op:    "replace",
				Path:  "/metadata/annotations/" + keyEscaped,
				Value: value,
			})
		}
	}
	return patch
}

func addContainers(target, added []corev1.Container, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Container{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func (whsvr *webhookServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "-- ok --")
}

func (whsvr *webhookServer) mutateHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Errorf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
	}

	glog.Info(body)

	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		http.Error(w, fmt.Sprintf("could not decode body: %v", err), http.StatusInternalServerError)
		return
	}

	admissionResponse := mutate(&ar)

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}
