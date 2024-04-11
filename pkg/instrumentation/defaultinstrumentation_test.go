// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aws/amazon-cloudwatch-agent-operator/apis/v1alpha1"
	"github.com/aws/amazon-cloudwatch-agent-operator/internal/manifests/collector/adapters"
)

func Test_getDefaultInstrumentation(t *testing.T) {
	httpInst := &v1alpha1.Instrumentation{
		Status: v1alpha1.InstrumentationStatus{},
		TypeMeta: metav1.TypeMeta{
			APIVersion: defaultAPIVersion,
			Kind:       defaultKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultInstrumenation,
			Namespace: defaultNamespace,
		},
		Spec: v1alpha1.InstrumentationSpec{
			Propagators: []v1alpha1.Propagator{
				v1alpha1.TraceContext,
				v1alpha1.Baggage,
				v1alpha1.B3,
				v1alpha1.XRay,
			},
			Java: v1alpha1.Java{
				Image: defaultJavaInstrumentationImage,
				Env: []corev1.EnvVar{
					{Name: "OTEL_SMP_ENABLED", Value: "true"}, //TODO: remove in favor of new name once safe
					{Name: "OTEL_AWS_APP_SIGNALS_ENABLED", Value: "true"},
					{Name: "OTEL_TRACES_SAMPLER_ARG", Value: "endpoint=http://cloudwatch-agent.amazon-cloudwatch:2000"},
					{Name: "OTEL_TRACES_SAMPLER", Value: "xray"},
					{Name: "OTEL_EXPORTER_OTLP_PROTOCOL", Value: "http/protobuf"},
					{Name: "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", Value: "http://cloudwatch-agent.amazon-cloudwatch:4316/v1/traces"},
					{Name: "OTEL_AWS_SMP_EXPORTER_ENDPOINT", Value: "http://cloudwatch-agent.amazon-cloudwatch:4316/v1/metrics"}, //TODO: remove in favor of new name once safe
					{Name: "OTEL_AWS_APP_SIGNALS_EXPORTER_ENDPOINT", Value: "http://cloudwatch-agent.amazon-cloudwatch:4316/v1/metrics"},
					{Name: "OTEL_METRICS_EXPORTER", Value: "none"},
					{Name: "OTEL_LOGS_EXPORTER", Value: "none"},
				},
			},
			Python: v1alpha1.Python{
				Image: defaultPythonInstrumentationImage,
				Env: []corev1.EnvVar{
					{Name: "OTEL_AWS_APP_SIGNALS_ENABLED", Value: "true"},
					{Name: "OTEL_TRACES_SAMPLER_ARG", Value: "endpoint=http://cloudwatch-agent.amazon-cloudwatch:2000"},
					{Name: "OTEL_TRACES_SAMPLER", Value: "xray"},
					{Name: "OTEL_EXPORTER_OTLP_PROTOCOL", Value: "http/protobuf"},
					{Name: "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", Value: "http://cloudwatch-agent.amazon-cloudwatch:4316/v1/traces"},
					{Name: "OTEL_AWS_APP_SIGNALS_EXPORTER_ENDPOINT", Value: "http://cloudwatch-agent.amazon-cloudwatch:4316/v1/metrics"},
					{Name: "OTEL_METRICS_EXPORTER", Value: "none"},
					{Name: "OTEL_PYTHON_DISTRO", Value: "aws_distro"},
					{Name: "OTEL_PYTHON_CONFIGURATOR", Value: "aws_configurator"},
					{Name: "OTEL_LOGS_EXPORTER", Value: "none"},
				},
			},
		},
	}
	httpsInst := &v1alpha1.Instrumentation{
		Status: v1alpha1.InstrumentationStatus{},
		TypeMeta: metav1.TypeMeta{
			APIVersion: defaultAPIVersion,
			Kind:       defaultKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultInstrumenation,
			Namespace: defaultNamespace,
		},
		Spec: v1alpha1.InstrumentationSpec{
			Propagators: []v1alpha1.Propagator{
				v1alpha1.TraceContext,
				v1alpha1.Baggage,
				v1alpha1.B3,
				v1alpha1.XRay,
			},
			Java: v1alpha1.Java{
				Image: defaultJavaInstrumentationImage,
				Env: []corev1.EnvVar{
					{Name: "OTEL_SMP_ENABLED", Value: "true"}, //TODO: remove in favor of new name once safe
					{Name: "OTEL_AWS_APP_SIGNALS_ENABLED", Value: "true"},
					{Name: "OTEL_TRACES_SAMPLER_ARG", Value: "endpoint=http://cloudwatch-agent.amazon-cloudwatch:2000"},
					{Name: "OTEL_TRACES_SAMPLER", Value: "xray"},
					{Name: "OTEL_EXPORTER_OTLP_PROTOCOL", Value: "http/protobuf"},
					{Name: "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", Value: "https://cloudwatch-agent.amazon-cloudwatch:4316/v1/traces"},
					{Name: "OTEL_AWS_SMP_EXPORTER_ENDPOINT", Value: "https://cloudwatch-agent.amazon-cloudwatch:4316/v1/metrics"}, //TODO: remove in favor of new name once safe
					{Name: "OTEL_AWS_APP_SIGNALS_EXPORTER_ENDPOINT", Value: "https://cloudwatch-agent.amazon-cloudwatch:4316/v1/metrics"},
					{Name: "OTEL_METRICS_EXPORTER", Value: "none"},
					{Name: "OTEL_LOGS_EXPORTER", Value: "none"},
				},
			},
			Python: v1alpha1.Python{
				Image: defaultPythonInstrumentationImage,
				Env: []corev1.EnvVar{
					{Name: "OTEL_AWS_APP_SIGNALS_ENABLED", Value: "true"},
					{Name: "OTEL_TRACES_SAMPLER_ARG", Value: "endpoint=http://cloudwatch-agent.amazon-cloudwatch:2000"},
					{Name: "OTEL_TRACES_SAMPLER", Value: "xray"},
					{Name: "OTEL_EXPORTER_OTLP_PROTOCOL", Value: "http/protobuf"},
					{Name: "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", Value: "https://cloudwatch-agent.amazon-cloudwatch:4316/v1/traces"},
					{Name: "OTEL_AWS_APP_SIGNALS_EXPORTER_ENDPOINT", Value: "https://cloudwatch-agent.amazon-cloudwatch:4316/v1/metrics"},
					{Name: "OTEL_METRICS_EXPORTER", Value: "none"},
					{Name: "OTEL_PYTHON_DISTRO", Value: "aws_distro"},
					{Name: "OTEL_PYTHON_CONFIGURATOR", Value: "aws_configurator"},
					{Name: "OTEL_LOGS_EXPORTER", Value: "none"},
				},
			},
		},
	}

	type args struct {
		agentConfig *adapters.CwaConfig
	}
	tests := []struct {
		name    string
		args    args
		want    *v1alpha1.Instrumentation
		wantErr bool
	}{
		{
			name: "http",
			args: args{
				agentConfig: &adapters.CwaConfig{
					Metrics: &adapters.Metrics{
						&adapters.MetricsCollected{
							AppSignals: &adapters.AppSignals{},
						},
					},
				},
			},
			want:    httpInst,
			wantErr: false,
		},
		{
			name: "https",
			args: args{
				agentConfig: &adapters.CwaConfig{
					Metrics: &adapters.Metrics{
						&adapters.MetricsCollected{
							AppSignals: &adapters.AppSignals{
								TLS: &adapters.TLS{
									CertFile: "some-cert",
									KeyFile:  "some-key",
								},
							},
						},
					},
				},
			},
			want:    httpsInst,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getDefaultInstrumentation(tt.args.agentConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("getDefaultInstrumentation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getDefaultInstrumentation() got = %v, want %v", got, tt.want)
			}
		})
	}
}
