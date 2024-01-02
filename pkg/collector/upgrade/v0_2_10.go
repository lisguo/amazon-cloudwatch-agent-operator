// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"github.com/aws/amazon-cloudwatch-agent-operator/apis/v1alpha1"
)

// this is our first version under otel/opentelemetry-collector.
func upgrade0_2_10(u VersionUpgrade, otelcol *v1alpha1.AmazonCloudWatchAgent) (*v1alpha1.AmazonCloudWatchAgent, error) {
	// this is a no-op, but serves to keep the skeleton here for the future versions
	return otelcol, nil
}
