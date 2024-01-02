// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"strings"
	"testing"

	"github.com/aws/amazon-cloudwatch-agent-operator/apis/v1alpha1"
	"github.com/aws/amazon-cloudwatch-agent-operator/internal/config"
	"github.com/aws/amazon-cloudwatch-agent-operator/internal/manifests"
	"github.com/aws/amazon-cloudwatch-agent-operator/pkg/featuregate"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	colfeaturegate "go.opentelemetry.io/collector/featuregate"
)

var (
	selectorLabels = map[string]string{
		"app.kubernetes.io/managed-by": "amazon-cloudwatch-agent-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "test.test",
	}
	basePolicy     = corev1.ServiceInternalTrafficPolicyCluster
	pathTypePrefix = networkingv1.PathTypePrefix
)

var (
	prometheusFeatureGate = featuregate.PrometheusOperatorIsAvailable.ID()
)

var (
	opampbridgeSelectorLabels = map[string]string{
		"app.kubernetes.io/managed-by": "amazon-cloudwatch-agent-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/component":  "opentelemetry-opamp-bridge",
		"app.kubernetes.io/instance":   "test.test",
	}
)

var (
	taSelectorLabels = map[string]string{
		"app.kubernetes.io/managed-by": "amazon-cloudwatch-agent-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/component":  "opentelemetry-targetallocator",
		"app.kubernetes.io/instance":   "test.test",
		"app.kubernetes.io/name":       "test-targetallocator",
	}
)

func TestBuildCollector(t *testing.T) {
	var goodConfig = `receivers:
  examplereceiver:
    endpoint: "0.0.0.0:12345"
exporters:
  logging:
service:
  pipelines:
    metrics:
      receivers: [examplereceiver]
      exporters: [logging]
`
	one := int32(1)
	type args struct {
		instance v1alpha1.AmazonCloudWatchAgent
	}
	tests := []struct {
		name    string
		args    args
		want    []client.Object
		wantErr bool
	}{
		{
			name: "base case",
			args: args{
				instance: v1alpha1.AmazonCloudWatchAgent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.AmazonCloudWatchAgentSpec{
						Replicas: &one,
						Mode:     "deployment",
						Image:    "test",
						Config:   goodConfig,
					},
				},
			},
			want: []client.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "amazon-cloudwatch-agent-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: map[string]string{
							"amazon-cloudwatch-agent-operator-config/sha256": "8188c85abd4aa5e9c798874b4f47d37f08d4778933154e3f7d60246510ed9dd2",
							"prometheus.io/path":                             "/metrics",
							"prometheus.io/port":                             "8888",
							"prometheus.io/scrape":                           "true",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: &one,
						Selector: &metav1.LabelSelector{
							MatchLabels: selectorLabels,
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/component":  "opentelemetry-collector",
									"app.kubernetes.io/instance":   "test.test",
									"app.kubernetes.io/managed-by": "amazon-cloudwatch-agent-operator",
									"app.kubernetes.io/name":       "test-collector",
									"app.kubernetes.io/part-of":    "opentelemetry",
									"app.kubernetes.io/version":    "latest",
								},
								Annotations: map[string]string{
									"amazon-cloudwatch-agent-operator-config/sha256": "8188c85abd4aa5e9c798874b4f47d37f08d4778933154e3f7d60246510ed9dd2",
									"prometheus.io/path":                             "/metrics",
									"prometheus.io/port":                             "8888",
									"prometheus.io/scrape":                           "true",
								},
							},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{
									{
										Name: "otc-internal",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-collector",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "collector.yaml",
														Path: "collector.yaml",
													},
												},
											},
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name:  "otc-container",
										Image: "test",
										Args: []string{
											"--config=/conf/collector.yaml",
										},
										Ports: []corev1.ContainerPort{
											{
												Name:          "examplereceiver",
												HostPort:      0,
												ContainerPort: 12345,
											},
											{
												Name:          "metrics",
												HostPort:      0,
												ContainerPort: 8888,
												Protocol:      "TCP",
											},
										},
										Env: []corev1.EnvVar{
											{
												Name: "POD_NAME",
												ValueFrom: &corev1.EnvVarSource{
													FieldRef: &corev1.ObjectFieldSelector{
														FieldPath: "metadata.name",
													},
												},
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "otc-internal",
												MountPath: "/conf",
											},
										},
									},
								},
								DNSPolicy:          "ClusterFirst",
								ServiceAccountName: "test-collector",
							},
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "amazon-cloudwatch-agent-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Data: map[string]string{
						"collector.yaml": "receivers:\n  examplereceiver:\n    endpoint: \"0.0.0.0:12345\"\nexporters:\n  logging:\nservice:\n  pipelines:\n    metrics:\n      receivers: [examplereceiver]\n      exporters: [logging]\n",
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "examplereceiver",
								Port: 12345,
							},
						},
						Selector:              selectorLabels,
						InternalTrafficPolicy: &basePolicy,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-headless",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":                          "opentelemetry-collector",
							"app.kubernetes.io/instance":                           "test.test",
							"app.kubernetes.io/managed-by":                         "opentelemetry-operator",
							"app.kubernetes.io/name":                               "test-collector",
							"app.kubernetes.io/part-of":                            "opentelemetry",
							"app.kubernetes.io/version":                            "latest",
							"operator.opentelemetry.io/collector-headless-service": "Exists",
						},
						Annotations: map[string]string{
							"service.beta.openshift.io/serving-cert-secret-name": "test-collector-headless-tls",
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "examplereceiver",
								Port: 12345,
							},
						},
						Selector:              selectorLabels,
						InternalTrafficPolicy: &basePolicy,
						ClusterIP:             "None",
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-monitoring",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector-monitoring",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "monitoring",
								Port: 8888,
							},
						},
						Selector: selectorLabels,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ingress",
			args: args{
				instance: v1alpha1.AmazonCloudWatchAgent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.AmazonCloudWatchAgentSpec{
						Replicas: &one,
						Mode:     "deployment",
						Image:    "test",
						Ingress: v1alpha1.Ingress{
							Type:     v1alpha1.IngressTypeNginx,
							Hostname: "example.com",
							Annotations: map[string]string{
								"something": "true",
							},
						},
						Config: goodConfig,
					},
				},
			},
			want: []client.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: map[string]string{
							"opentelemetry-operator-config/sha256": "8188c85abd4aa5e9c798874b4f47d37f08d4778933154e3f7d60246510ed9dd2",
							"prometheus.io/path":                   "/metrics",
							"prometheus.io/port":                   "8888",
							"prometheus.io/scrape":                 "true",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: &one,
						Selector: &metav1.LabelSelector{
							MatchLabels: selectorLabels,
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/component":  "opentelemetry-collector",
									"app.kubernetes.io/instance":   "test.test",
									"app.kubernetes.io/managed-by": "opentelemetry-operator",
									"app.kubernetes.io/name":       "test-collector",
									"app.kubernetes.io/part-of":    "opentelemetry",
									"app.kubernetes.io/version":    "latest",
								},
								Annotations: map[string]string{
									"opentelemetry-operator-config/sha256": "8188c85abd4aa5e9c798874b4f47d37f08d4778933154e3f7d60246510ed9dd2",
									"prometheus.io/path":                   "/metrics",
									"prometheus.io/port":                   "8888",
									"prometheus.io/scrape":                 "true",
								},
							},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{
									{
										Name: "otc-internal",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-collector",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "collector.yaml",
														Path: "collector.yaml",
													},
												},
											},
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name:  "otc-container",
										Image: "test",
										Args: []string{
											"--config=/conf/collector.yaml",
										},
										Ports: []corev1.ContainerPort{
											{
												Name:          "examplereceiver",
												HostPort:      0,
												ContainerPort: 12345,
											},
											{
												Name:          "metrics",
												HostPort:      0,
												ContainerPort: 8888,
												Protocol:      "TCP",
											},
										},
										Env: []corev1.EnvVar{
											{
												Name: "POD_NAME",
												ValueFrom: &corev1.EnvVarSource{
													FieldRef: &corev1.ObjectFieldSelector{
														FieldPath: "metadata.name",
													},
												},
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "otc-internal",
												MountPath: "/conf",
											},
										},
									},
								},
								DNSPolicy:          "ClusterFirst",
								ServiceAccountName: "test-collector",
							},
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Data: map[string]string{
						"collector.yaml": "receivers:\n  examplereceiver:\n    endpoint: \"0.0.0.0:12345\"\nexporters:\n  logging:\nservice:\n  pipelines:\n    metrics:\n      receivers: [examplereceiver]\n      exporters: [logging]\n",
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "examplereceiver",
								Port: 12345,
							},
						},
						Selector:              selectorLabels,
						InternalTrafficPolicy: &basePolicy,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-headless",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":                          "opentelemetry-collector",
							"app.kubernetes.io/instance":                           "test.test",
							"app.kubernetes.io/managed-by":                         "opentelemetry-operator",
							"app.kubernetes.io/name":                               "test-collector",
							"app.kubernetes.io/part-of":                            "opentelemetry",
							"app.kubernetes.io/version":                            "latest",
							"operator.opentelemetry.io/collector-headless-service": "Exists",
						},
						Annotations: map[string]string{
							"service.beta.openshift.io/serving-cert-secret-name": "test-collector-headless-tls",
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "examplereceiver",
								Port: 12345,
							},
						},
						Selector:              selectorLabels,
						InternalTrafficPolicy: &basePolicy,
						ClusterIP:             "None",
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-monitoring",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector-monitoring",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "monitoring",
								Port: 8888,
							},
						},
						Selector: selectorLabels,
					},
				},
				&networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ingress",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-ingress",
						},
						Annotations: map[string]string{
							"something": "true",
						},
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: "example.com",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path:     "/examplereceiver",
												PathType: &pathTypePrefix,
												Backend: networkingv1.IngressBackend{
													Service: &networkingv1.IngressServiceBackend{
														Name: "test-collector",
														Port: networkingv1.ServiceBackendPort{
															Name: "examplereceiver",
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "specified service account case",
			args: args{
				instance: v1alpha1.AmazonCloudWatchAgent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.AmazonCloudWatchAgentSpec{
						Replicas:       &one,
						Mode:           "deployment",
						Image:          "test",
						Config:         goodConfig,
						ServiceAccount: "my-special-sa",
					},
				},
			},
			want: []client.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: map[string]string{
							"opentelemetry-operator-config/sha256": "8188c85abd4aa5e9c798874b4f47d37f08d4778933154e3f7d60246510ed9dd2",
							"prometheus.io/path":                   "/metrics",
							"prometheus.io/port":                   "8888",
							"prometheus.io/scrape":                 "true",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: &one,
						Selector: &metav1.LabelSelector{
							MatchLabels: selectorLabels,
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/component":  "opentelemetry-collector",
									"app.kubernetes.io/instance":   "test.test",
									"app.kubernetes.io/managed-by": "opentelemetry-operator",
									"app.kubernetes.io/name":       "test-collector",
									"app.kubernetes.io/part-of":    "opentelemetry",
									"app.kubernetes.io/version":    "latest",
								},
								Annotations: map[string]string{
									"opentelemetry-operator-config/sha256": "8188c85abd4aa5e9c798874b4f47d37f08d4778933154e3f7d60246510ed9dd2",
									"prometheus.io/path":                   "/metrics",
									"prometheus.io/port":                   "8888",
									"prometheus.io/scrape":                 "true",
								},
							},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{
									{
										Name: "otc-internal",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-collector",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "collector.yaml",
														Path: "collector.yaml",
													},
												},
											},
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name:  "otc-container",
										Image: "test",
										Args: []string{
											"--config=/conf/collector.yaml",
										},
										Ports: []corev1.ContainerPort{
											{
												Name:          "examplereceiver",
												HostPort:      0,
												ContainerPort: 12345,
											},
											{
												Name:          "metrics",
												HostPort:      0,
												ContainerPort: 8888,
												Protocol:      "TCP",
											},
										},
										Env: []corev1.EnvVar{
											{
												Name: "POD_NAME",
												ValueFrom: &corev1.EnvVarSource{
													FieldRef: &corev1.ObjectFieldSelector{
														FieldPath: "metadata.name",
													},
												},
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "otc-internal",
												MountPath: "/conf",
											},
										},
									},
								},
								DNSPolicy:          "ClusterFirst",
								ServiceAccountName: "my-special-sa",
							},
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Data: map[string]string{
						"collector.yaml": "receivers:\n  examplereceiver:\n    endpoint: \"0.0.0.0:12345\"\nexporters:\n  logging:\nservice:\n  pipelines:\n    metrics:\n      receivers: [examplereceiver]\n      exporters: [logging]\n",
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "examplereceiver",
								Port: 12345,
							},
						},
						Selector:              selectorLabels,
						InternalTrafficPolicy: &basePolicy,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-headless",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":                          "opentelemetry-collector",
							"app.kubernetes.io/instance":                           "test.test",
							"app.kubernetes.io/managed-by":                         "opentelemetry-operator",
							"app.kubernetes.io/name":                               "test-collector",
							"app.kubernetes.io/part-of":                            "opentelemetry",
							"app.kubernetes.io/version":                            "latest",
							"operator.opentelemetry.io/collector-headless-service": "Exists",
						},
						Annotations: map[string]string{
							"service.beta.openshift.io/serving-cert-secret-name": "test-collector-headless-tls",
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "examplereceiver",
								Port: 12345,
							},
						},
						Selector:              selectorLabels,
						InternalTrafficPolicy: &basePolicy,
						ClusterIP:             "None",
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-monitoring",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector-monitoring",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "monitoring",
								Port: 8888,
							},
						},
						Selector: selectorLabels,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.New(
				config.WithCollectorImage("default-collector"),
				config.WithTargetAllocatorImage("default-ta-allocator"),
			)
			params := manifests.Params{
				Log:     logr.Discard(),
				Config:  cfg,
				OtelCol: tt.args.instance,
			}
			got, err := BuildCollector(params)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)

		})
	}
}

func TestBuildTargetAllocator(t *testing.T) {
	var goodConfig = `
receivers:
  prometheus:
    config:
      scrape_configs:
      - job_name: 'example'
        relabel_configs:
        - source_labels: ['__meta_service_id']
          target_label: 'job'
          replacement: 'my_service_$$1'
        - source_labels: ['__meta_service_name']
          target_label: 'instance'
          replacement: '$1'
        metric_relabel_configs:
        - source_labels: ['job']
          target_label: 'job'
          replacement: '$$1_$2'
exporters:
  logging:
service:
  pipelines:
    metrics:
      receivers: [prometheus]
      exporters: [logging]
`
	one := int32(1)
	type args struct {
		instance v1alpha1.AmazonCloudWatchAgent
	}
	tests := []struct {
		name         string
		args         args
		want         []client.Object
		featuregates []string
		wantErr      bool
	}{
		{
			name: "base case",
			args: args{
				instance: v1alpha1.AmazonCloudWatchAgent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.AmazonCloudWatchAgentSpec{
						Replicas: &one,
						Mode:     "statefulset",
						Image:    "test",
						Config:   goodConfig,
						TargetAllocator: v1alpha1.OpenTelemetryTargetAllocator{
							Enabled: true,
							PrometheusCR: v1alpha1.OpenTelemetryTargetAllocatorPrometheusCR{
								Enabled: true,
							},
						},
					},
				},
			},
			want: []client.Object{
				&appsv1.StatefulSet{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: map[string]string{
							"opentelemetry-operator-config/sha256": "5fe4d7d7faf3247bd7ec88688c9f73618f9ab8e170362bd7203c54e7a2f5cec0",
							"prometheus.io/path":                   "/metrics",
							"prometheus.io/port":                   "8888",
							"prometheus.io/scrape":                 "true",
						},
					},
					Spec: appsv1.StatefulSetSpec{
						ServiceName: "test-collector",
						Replicas:    &one,
						Selector: &metav1.LabelSelector{
							MatchLabels: selectorLabels,
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/component":  "opentelemetry-collector",
									"app.kubernetes.io/instance":   "test.test",
									"app.kubernetes.io/managed-by": "opentelemetry-operator",
									"app.kubernetes.io/name":       "test-collector",
									"app.kubernetes.io/part-of":    "opentelemetry",
									"app.kubernetes.io/version":    "latest",
								},
								Annotations: map[string]string{
									"opentelemetry-operator-config/sha256": "5fe4d7d7faf3247bd7ec88688c9f73618f9ab8e170362bd7203c54e7a2f5cec0",
									"prometheus.io/path":                   "/metrics",
									"prometheus.io/port":                   "8888",
									"prometheus.io/scrape":                 "true",
								},
							},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{
									{
										Name: "otc-internal",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-collector",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "collector.yaml",
														Path: "collector.yaml",
													},
												},
											},
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name:  "otc-container",
										Image: "test",
										Args: []string{
											"--config=/conf/collector.yaml",
										},
										Env: []corev1.EnvVar{
											{
												Name: "POD_NAME",
												ValueFrom: &corev1.EnvVarSource{
													FieldRef: &corev1.ObjectFieldSelector{
														FieldPath: "metadata.name",
													},
												},
											},
											{
												Name:  "SHARD",
												Value: "0",
											},
										},
										Ports: []corev1.ContainerPort{
											{
												Name:          "metrics",
												HostPort:      0,
												ContainerPort: 8888,
												Protocol:      "TCP",
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "otc-internal",
												MountPath: "/conf",
											},
										},
									},
								},
								DNSPolicy:          "ClusterFirst",
								ServiceAccountName: "test-collector",
							},
						},
						PodManagementPolicy: "Parallel",
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Data: map[string]string{
						"collector.yaml": "exporters:\n  logging: null\nreceivers:\n  prometheus:\n    config: {}\n    target_allocator:\n      collector_id: ${POD_NAME}\n      endpoint: http://test-targetallocator:80\n      interval: 30s\nservice:\n  pipelines:\n    metrics:\n      exporters:\n      - logging\n      receivers:\n      - prometheus\n",
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-monitoring",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector-monitoring",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "monitoring",
								Port: 8888,
							},
						},
						Selector: selectorLabels,
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Data: map[string]string{
						"targetallocator.yaml": "allocation_strategy: least-weighted\nconfig:\n  scrape_configs:\n  - job_name: example\n    metric_relabel_configs:\n    - replacement: $1_$2\n      source_labels:\n      - job\n      target_label: job\n    relabel_configs:\n    - replacement: my_service_$1\n      source_labels:\n      - __meta_service_id\n      target_label: job\n    - replacement: $1\n      source_labels:\n      - __meta_service_name\n      target_label: instance\nlabel_selector:\n  app.kubernetes.io/component: opentelemetry-collector\n  app.kubernetes.io/instance: test.test\n  app.kubernetes.io/managed-by: opentelemetry-operator\n  app.kubernetes.io/part-of: opentelemetry\n",
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: taSelectorLabels,
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/component":  "opentelemetry-targetallocator",
									"app.kubernetes.io/instance":   "test.test",
									"app.kubernetes.io/managed-by": "opentelemetry-operator",
									"app.kubernetes.io/name":       "test-targetallocator",
									"app.kubernetes.io/part-of":    "opentelemetry",
									"app.kubernetes.io/version":    "latest",
								},
								Annotations: map[string]string{
									"opentelemetry-targetallocator-config/hash": "4d1911fd40106e9e2dd3d928f067a6c8c9eab0c569f737ba3701c6f5a9aad6d7",
								},
							},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{
									{
										Name: "ta-internal",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-targetallocator",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "targetallocator.yaml",
														Path: "targetallocator.yaml",
													},
												},
											},
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name:  "ta-container",
										Image: "default-ta-allocator",
										Env: []corev1.EnvVar{
											{
												Name: "OTELCOL_NAMESPACE",
												ValueFrom: &corev1.EnvVarSource{
													FieldRef: &corev1.ObjectFieldSelector{
														FieldPath: "metadata.namespace",
													},
												},
											},
										},
										Args: []string{
											"--enable-prometheus-cr-watcher",
										},
										Ports: []corev1.ContainerPort{
											{
												Name:          "http",
												HostPort:      0,
												ContainerPort: 8080,
												Protocol:      "TCP",
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "ta-internal",
												MountPath: "/conf",
											},
										},
										LivenessProbe: &corev1.Probe{
											ProbeHandler: corev1.ProbeHandler{
												HTTPGet: &corev1.HTTPGetAction{
													Path: "/livez",
													Port: intstr.FromInt(8080),
												},
											},
										},
										ReadinessProbe: &corev1.Probe{
											ProbeHandler: corev1.ProbeHandler{
												HTTPGet: &corev1.HTTPGetAction{
													Path: "/readyz",
													Port: intstr.FromInt(8080),
												},
											},
										},
									},
								},
								DNSPolicy:          "",
								ServiceAccountName: "test-targetallocator",
							},
						},
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "targetallocation",
								Port: 80,
								TargetPort: intstr.IntOrString{
									Type:   1,
									StrVal: "http",
								},
							},
						},
						Selector: taSelectorLabels,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "enable metrics case",
			args: args{
				instance: v1alpha1.AmazonCloudWatchAgent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.AmazonCloudWatchAgentSpec{
						Replicas: &one,
						Mode:     "statefulset",
						Image:    "test",
						Config:   goodConfig,
						TargetAllocator: v1alpha1.OpenTelemetryTargetAllocator{
							Enabled: true,
							PrometheusCR: v1alpha1.OpenTelemetryTargetAllocatorPrometheusCR{
								Enabled: true,
							},
							Observability: v1alpha1.ObservabilitySpec{
								Metrics: v1alpha1.MetricsConfigSpec{
									EnableMetrics: true,
								},
							},
						},
					},
				},
			},
			want: []client.Object{
				&appsv1.StatefulSet{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: map[string]string{
							"opentelemetry-operator-config/sha256": "5fe4d7d7faf3247bd7ec88688c9f73618f9ab8e170362bd7203c54e7a2f5cec0",
							"prometheus.io/path":                   "/metrics",
							"prometheus.io/port":                   "8888",
							"prometheus.io/scrape":                 "true",
						},
					},
					Spec: appsv1.StatefulSetSpec{
						ServiceName: "test-collector",
						Replicas:    &one,
						Selector: &metav1.LabelSelector{
							MatchLabels: selectorLabels,
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/component":  "opentelemetry-collector",
									"app.kubernetes.io/instance":   "test.test",
									"app.kubernetes.io/managed-by": "opentelemetry-operator",
									"app.kubernetes.io/name":       "test-collector",
									"app.kubernetes.io/part-of":    "opentelemetry",
									"app.kubernetes.io/version":    "latest",
								},
								Annotations: map[string]string{
									"opentelemetry-operator-config/sha256": "5fe4d7d7faf3247bd7ec88688c9f73618f9ab8e170362bd7203c54e7a2f5cec0",
									"prometheus.io/path":                   "/metrics",
									"prometheus.io/port":                   "8888",
									"prometheus.io/scrape":                 "true",
								},
							},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{
									{
										Name: "otc-internal",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-collector",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "collector.yaml",
														Path: "collector.yaml",
													},
												},
											},
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name:  "otc-container",
										Image: "test",
										Args: []string{
											"--config=/conf/collector.yaml",
										},
										Env: []corev1.EnvVar{
											{
												Name: "POD_NAME",
												ValueFrom: &corev1.EnvVarSource{
													FieldRef: &corev1.ObjectFieldSelector{
														FieldPath: "metadata.name",
													},
												},
											},
											{
												Name:  "SHARD",
												Value: "0",
											},
										},
										Ports: []corev1.ContainerPort{
											{
												Name:          "metrics",
												HostPort:      0,
												ContainerPort: 8888,
												Protocol:      "TCP",
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "otc-internal",
												MountPath: "/conf",
											},
										},
									},
								},
								DNSPolicy:          "ClusterFirst",
								ServiceAccountName: "test-collector",
							},
						},
						PodManagementPolicy: "Parallel",
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Data: map[string]string{
						"collector.yaml": "exporters:\n  logging: null\nreceivers:\n  prometheus:\n    config: {}\n    target_allocator:\n      collector_id: ${POD_NAME}\n      endpoint: http://test-targetallocator:80\n      interval: 30s\nservice:\n  pipelines:\n    metrics:\n      exporters:\n      - logging\n      receivers:\n      - prometheus\n",
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector-monitoring",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-collector-monitoring",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "monitoring",
								Port: 8888,
							},
						},
						Selector: selectorLabels,
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Data: map[string]string{
						"targetallocator.yaml": "allocation_strategy: least-weighted\nconfig:\n  scrape_configs:\n  - job_name: example\n    metric_relabel_configs:\n    - replacement: $1_$2\n      source_labels:\n      - job\n      target_label: job\n    relabel_configs:\n    - replacement: my_service_$1\n      source_labels:\n      - __meta_service_id\n      target_label: job\n    - replacement: $1\n      source_labels:\n      - __meta_service_name\n      target_label: instance\nlabel_selector:\n  app.kubernetes.io/component: opentelemetry-collector\n  app.kubernetes.io/instance: test.test\n  app.kubernetes.io/managed-by: opentelemetry-operator\n  app.kubernetes.io/part-of: opentelemetry\n",
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: taSelectorLabels,
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/component":  "opentelemetry-targetallocator",
									"app.kubernetes.io/instance":   "test.test",
									"app.kubernetes.io/managed-by": "opentelemetry-operator",
									"app.kubernetes.io/name":       "test-targetallocator",
									"app.kubernetes.io/part-of":    "opentelemetry",
									"app.kubernetes.io/version":    "latest",
								},
								Annotations: map[string]string{
									"opentelemetry-targetallocator-config/hash": "4d1911fd40106e9e2dd3d928f067a6c8c9eab0c569f737ba3701c6f5a9aad6d7",
								},
							},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{
									{
										Name: "ta-internal",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "test-targetallocator",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "targetallocator.yaml",
														Path: "targetallocator.yaml",
													},
												},
											},
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name:  "ta-container",
										Image: "default-ta-allocator",
										Env: []corev1.EnvVar{
											{
												Name: "OTELCOL_NAMESPACE",
												ValueFrom: &corev1.EnvVarSource{
													FieldRef: &corev1.ObjectFieldSelector{
														FieldPath: "metadata.namespace",
													},
												},
											},
										},
										Args: []string{
											"--enable-prometheus-cr-watcher",
										},
										Ports: []corev1.ContainerPort{
											{
												Name:          "http",
												HostPort:      0,
												ContainerPort: 8080,
												Protocol:      "TCP",
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "ta-internal",
												MountPath: "/conf",
											},
										},
										LivenessProbe: &corev1.Probe{
											ProbeHandler: corev1.ProbeHandler{
												HTTPGet: &corev1.HTTPGetAction{
													Path: "/livez",
													Port: intstr.FromInt(8080),
												},
											},
										},
										ReadinessProbe: &corev1.Probe{
											ProbeHandler: corev1.ProbeHandler{
												HTTPGet: &corev1.HTTPGetAction{
													Path: "/readyz",
													Port: intstr.FromInt(8080),
												},
											},
										},
									},
								},
								DNSPolicy:          "",
								ServiceAccountName: "test-targetallocator",
							},
						},
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "targetallocation",
								Port: 80,
								TargetPort: intstr.IntOrString{
									Type:   1,
									StrVal: "http",
								},
							},
						},
						Selector: taSelectorLabels,
					},
				},
				&monitoringv1.ServiceMonitor{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-targetallocator",
						Namespace: "test",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "test.test",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/name":       "test-targetallocator",
							"app.kubernetes.io/part-of":    "opentelemetry",
							"app.kubernetes.io/version":    "latest",
						},
						Annotations: nil,
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						Endpoints: []monitoringv1.Endpoint{
							monitoringv1.Endpoint{Port: "targetallocation"},
						},
						Selector: v1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-targetallocator",
								"app.kubernetes.io/instance":   "test.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/name":       "test-targetallocator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
						NamespaceSelector: monitoringv1.NamespaceSelector{
							MatchNames: []string{"test"},
						},
					},
				},
			},
			wantErr:      false,
			featuregates: []string{prometheusFeatureGate},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.New(
				config.WithCollectorImage("default-collector"),
				config.WithTargetAllocatorImage("default-ta-allocator"),
			)
			params := manifests.Params{
				Log:     logr.Discard(),
				Config:  cfg,
				OtelCol: tt.args.instance,
			}
			if len(tt.featuregates) > 0 {
				fg := strings.Join(tt.featuregates, ",")
				flagset := featuregate.Flags(colfeaturegate.GlobalRegistry())
				if err := flagset.Set(featuregate.FeatureGatesFlag, fg); err != nil {
					t.Errorf("featuregate setting error = %v", err)
					return
				}
			}
			got, err := BuildCollector(params)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)

		})
	}
}
