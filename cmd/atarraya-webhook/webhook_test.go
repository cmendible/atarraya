package main

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_webhookServer_mutateContainers(t *testing.T) {
	type args struct {
		containers   []corev1.Container
		keyvaultName string
	}
	tests := []struct {
		name    string
		mwh     *webhookServer
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Mutation adds atarraya",
			args: args{
				containers: []corev1.Container{
					corev1.Container{
						Name:    "alpine",
						Image:   "alpine",
						Command: []string{"/bin/sh"},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.mwh.mutateContainers(tt.args.containers, tt.args.keyvaultName)
			if (err != nil) != tt.wantErr {
				t.Errorf("webhookServer.mutateContainers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("webhookServer.mutateContainers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getInitContainers(t *testing.T) {
	tests := []struct {
		name string
		want []corev1.Container
	}{
		{
			name: "getInitContainers returns expected values",
			want: []corev1.Container{
				{
					Name:            "az-atarraya-init",
					Image:           "cmendibl3/atarraya:latest",
					ImagePullPolicy: corev1.PullAlways,
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getInitContainers(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getInitContainers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getVolumes(t *testing.T) {
	tests := []struct {
		name string
		want []corev1.Volume
	}{
		{
			name: "getVolumes returns expected values",
			want: []corev1.Volume{
				{
					Name: "atarraya-volume",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{
							Medium: corev1.StorageMediumMemory,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getVolumes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getVolumes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_webhookServer_atarrayaMutator(t *testing.T) {
	type args struct {
		ctx context.Context
		obj metav1.Object
	}
	tests := []struct {
		name    string
		mwh     *webhookServer
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.mwh.atarrayaMutator(tt.args.ctx, tt.args.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("webhookServer.atarrayaMutator() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("webhookServer.atarrayaMutator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_webhookServer_mutatePod(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name    string
		mwh     *webhookServer
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.mwh.mutatePod(tt.args.pod); (err != nil) != tt.wantErr {
				t.Errorf("webhookServer.mutatePod() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_webhookServer_healthHandler(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		mwh  *webhookServer
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mwh.healthHandler(tt.args.w, tt.args.r)
		})
	}
}
