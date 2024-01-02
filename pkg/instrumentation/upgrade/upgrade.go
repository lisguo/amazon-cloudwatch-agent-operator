// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	featuregate2 "go.opentelemetry.io/collector/featuregate"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aws/amazon-cloudwatch-agent-operator/apis/v1alpha1"
	"github.com/aws/amazon-cloudwatch-agent-operator/pkg/constants"
	"github.com/aws/amazon-cloudwatch-agent-operator/pkg/featuregate"
)

var (
	defaultAnnotationToGate = map[string]*featuregate2.Gate{
		constants.AnnotationDefaultAutoInstrumentationJava:        featuregate.EnableJavaAutoInstrumentationSupport,
		constants.AnnotationDefaultAutoInstrumentationNodeJS:      featuregate.EnableNodeJSAutoInstrumentationSupport,
		constants.AnnotationDefaultAutoInstrumentationPython:      featuregate.EnablePythonAutoInstrumentationSupport,
		constants.AnnotationDefaultAutoInstrumentationDotNet:      featuregate.EnableDotnetAutoInstrumentationSupport,
		constants.AnnotationDefaultAutoInstrumentationGo:          featuregate.EnableGoAutoInstrumentationSupport,
		constants.AnnotationDefaultAutoInstrumentationApacheHttpd: featuregate.EnableApacheHTTPAutoInstrumentationSupport,
		constants.AnnotationDefaultAutoInstrumentationNginx:       featuregate.EnableNginxAutoInstrumentationSupport,
	}
)

type InstrumentationUpgrade struct {
	Client                     client.Client
	Logger                     logr.Logger
	Recorder                   record.EventRecorder
	DefaultAutoInstJava        string
	DefaultAutoInstNodeJS      string
	DefaultAutoInstPython      string
	DefaultAutoInstDotNet      string
	DefaultAutoInstApacheHttpd string
	DefaultAutoInstNginx       string
	DefaultAutoInstGo          string
}

// +kubebuilder:rbac:groups=cloudwatch.aws.amazon.com,resources=instrumentations,verbs=get;list;watch;update;patch

// ManagedInstances upgrades managed instances by the amazon-cloudwatch-agent-operator.
func (u *InstrumentationUpgrade) ManagedInstances(ctx context.Context) error {
	u.Logger.Info("looking for managed Instrumentation instances to upgrade")

	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/managed-by": "amazon-cloudwatch-agent-operator",
		}),
	}
	list := &v1alpha1.InstrumentationList{}
	if err := u.Client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for i := range list.Items {
		toUpgrade := list.Items[i]
		upgraded := u.upgrade(ctx, toUpgrade)
		if !reflect.DeepEqual(upgraded, toUpgrade) {
			// use update instead of patch because the patch does not upgrade annotations
			if err := u.Client.Update(ctx, upgraded); err != nil {
				u.Logger.Error(err, "failed to apply changes to instance", "name", upgraded.Name, "namespace", upgraded.Namespace)
				continue
			}
		}
	}

	if len(list.Items) == 0 {
		u.Logger.Info("no instances to upgrade")
	}
	return nil
}

func (u *InstrumentationUpgrade) upgrade(_ context.Context, inst v1alpha1.Instrumentation) *v1alpha1.Instrumentation {
	upgraded := inst.DeepCopy()
	for annotation, gate := range defaultAnnotationToGate {
		autoInst := upgraded.Annotations[annotation]
		if autoInst != "" {
			if gate.IsEnabled() {
				switch annotation {
				case constants.AnnotationDefaultAutoInstrumentationJava:
					if inst.Spec.Java.Image == autoInst {
						upgraded.Spec.Java.Image = u.DefaultAutoInstJava
						upgraded.Annotations[annotation] = u.DefaultAutoInstJava
					}
				}
			} else {
				u.Logger.Error(nil, "autoinstrumentation not enabled for this language", "flag", gate.ID())
				u.Recorder.Event(upgraded, "Warning", "InstrumentationUpgradeRejected", fmt.Sprintf("support for is not enabled for %s", gate.ID()))
			}
		}
	}
	return upgraded
}
