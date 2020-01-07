package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

const (
	requiredAnnotation  = "atarraya/keyvault"
	statusAnnotationKey = "atarraya/status"
	// StatusInjected is the annotation value for /status that indicates an injection was already performed on this pod
	statusInjected = "injected"
)

type webhookServer struct {
	Server *http.Server
}

func (mwh *webhookServer) mutateContainers(containers []corev1.Container, keyvaultName string) (bool, error) {
	mutated := false

	for i, container := range containers {
		mutated = true

		args := container.Command

		args = append(args, container.Args...)

		container.Command = []string{"/atarraya/atarraya"}
		container.Args = args

		container.VolumeMounts = append(container.VolumeMounts, []corev1.VolumeMount{
			{
				Name:      "atarraya-volume",
				MountPath: "/atarraya/",
			},
		}...)

		container.Env = append(container.Env, []corev1.EnvVar{
			{
				Name:  "ATARRAYA_AZURE_KEYVAULT_NAME",
				Value: keyvaultName,
			},
		}...)

		containers[i] = container
	}

	return mutated, nil
}

func getInitContainers() []corev1.Container {
	return []corev1.Container{
		{
			Name:            "az-atarraya-init",
			Image:           viper.GetString("atrraya_image"),
			ImagePullPolicy: corev1.PullPolicy(viper.GetString("atrraya_image_pull_policy")),
			Command:         []string{"sh", "-c", "cp /usr/local/bin/atarraya /atarraya/"},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "atarraya-volume",
					MountPath: "/atarraya/",
				},
			},

			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("32Mi"),
				},
			},
		},
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "atarraya-volume",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		},
	}
}

func (mwh *webhookServer) atarrayaMutator(ctx context.Context, obj metav1.Object) (bool, error) {
	switch v := obj.(type) {
	case *corev1.Pod:
		return false, mwh.mutatePod(v)
	default:
		return false, nil
	}
}

func (mwh *webhookServer) mutatePod(pod *corev1.Pod) error {
	metadata := pod.ObjectMeta

	annotations := metadata.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	skip := false

	status, ok := annotations[statusAnnotationKey]
	if ok && strings.ToLower(status) == statusInjected {
		glog.Infof("Pod %s/%s annotation %s=%s indicates injection already satisfied, skipping", metadata.Namespace, metadata.Name, statusAnnotationKey, status)
		skip = true
	}

	// determine whether to perform mutation based on annotation for the target resource
	keyvaultName, ok := annotations[requiredAnnotation]
	if !ok {
		glog.Infof("Pod %s/%s annotation %s is missing, skipping injection", metadata.Namespace, metadata.Name, requiredAnnotation)
		skip = true
	}

	if !skip {
		mwh.mutateContainers(pod.Spec.Containers, keyvaultName)
		pod.Spec.InitContainers = append(getInitContainers(), pod.Spec.InitContainers...)
		pod.Spec.Volumes = append(pod.Spec.Volumes, getVolumes()...)
		pod.ObjectMeta.Annotations[statusAnnotationKey] = statusInjected
	}

	return nil
}

func (mwh *webhookServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "-- ok --")
}
