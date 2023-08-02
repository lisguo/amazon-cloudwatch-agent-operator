// Package config contains the operator's runtime configuration.
package config

import (
	"sync"
	"time"

	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/aws/amazon-cloudwatch-agent-operator/internal/version"
	"github.com/aws/amazon-cloudwatch-agent-operator/pkg/autodetect"
)

const (
	defaultAutoDetectFrequency           = 5 * time.Second
	defaultTargetAllocatorConfigMapEntry = "targetallocator.yaml"
	defaultCloudWatchAgentConfigMapEntry = "cwagentconfig.json"
)

// Config holds the static configuration for this operator.
type Config struct {
	autoDetect                          autodetect.AutoDetect
	logger                              logr.Logger
	targetAllocatorImage                string
	operatorOpAMPBridgeImage            string
	autoInstrumentationPythonImage      string
	collectorImage                      string
	collectorConfigMapEntry             string
	autoInstrumentationDotNetImage      string
	autoInstrumentationGoImage          string
	autoInstrumentationApacheHttpdImage string
	autoInstrumentationNginxImage       string
	targetAllocatorConfigMapEntry       string
	autoInstrumentationNodeJSImage      string
	autoInstrumentationJavaImage        string
	onOpenShiftRoutesChange             changeHandler
	labelsFilter                        []string
	openshiftRoutes                     openshiftRoutesStore
	autoDetectFrequency                 time.Duration
}

// New constructs a new configuration based on the given options.
func New(opts ...Option) Config {
	// initialize with the default values
	o := options{
		autoDetectFrequency:           defaultAutoDetectFrequency,
		collectorConfigMapEntry:       defaultCloudWatchAgentConfigMapEntry,
		targetAllocatorConfigMapEntry: defaultTargetAllocatorConfigMapEntry,
		logger:                        logf.Log.WithName("config"),
		openshiftRoutes:               newOpenShiftRoutesWrapper(),
		version:                       version.Get(),
		onOpenShiftRoutesChange:       newOnChange(),
	}
	for _, opt := range opts {
		opt(&o)
	}

	return Config{
		autoDetect:                          o.autoDetect,
		autoDetectFrequency:                 o.autoDetectFrequency,
		collectorImage:                      o.collectorImage,
		collectorConfigMapEntry:             o.collectorConfigMapEntry,
		targetAllocatorImage:                o.targetAllocatorImage,
		operatorOpAMPBridgeImage:            o.operatorOpAMPBridgeImage,
		targetAllocatorConfigMapEntry:       o.targetAllocatorConfigMapEntry,
		logger:                              o.logger,
		openshiftRoutes:                     o.openshiftRoutes,
		onOpenShiftRoutesChange:             o.onOpenShiftRoutesChange,
		autoInstrumentationJavaImage:        o.autoInstrumentationJavaImage,
		autoInstrumentationNodeJSImage:      o.autoInstrumentationNodeJSImage,
		autoInstrumentationPythonImage:      o.autoInstrumentationPythonImage,
		autoInstrumentationDotNetImage:      o.autoInstrumentationDotNetImage,
		autoInstrumentationGoImage:          o.autoInstrumentationGoImage,
		autoInstrumentationApacheHttpdImage: o.autoInstrumentationApacheHttpdImage,
		autoInstrumentationNginxImage:       o.autoInstrumentationNginxImage,
		labelsFilter:                        o.labelsFilter,
	}
}

// StartAutoDetect attempts to automatically detect relevant information for this operator. This will block until the first
// run is executed and will schedule periodic updates.
func (c *Config) StartAutoDetect() error {
	err := c.AutoDetect()
	go c.periodicAutoDetect()

	return err
}

func (c *Config) periodicAutoDetect() {
	ticker := time.NewTicker(c.autoDetectFrequency)

	for range ticker.C {
		if err := c.AutoDetect(); err != nil {
			c.logger.Info("auto-detection failed", "error", err)
		}
	}
}

// AutoDetect attempts to automatically detect relevant information for this operator.
func (c *Config) AutoDetect() error {
	c.logger.V(2).Info("auto-detecting the configuration based on the environment")

	ora, err := c.autoDetect.OpenShiftRoutesAvailability()
	if err != nil {
		return err
	}

	if c.openshiftRoutes.Get() != ora {
		c.logger.V(1).Info("openshift routes detected", "available", ora)
		c.openshiftRoutes.Set(ora)
		if err = c.onOpenShiftRoutesChange.Do(); err != nil {
			// Don't fail if the callback failed, as auto-detection itself worked.
			c.logger.Error(err, "configuration change notification failed for callback")
		}
	}

	return nil
}

// CollectorImage represents the flag to override the OpenTelemetry Collector container image.
func (c *Config) CollectorImage() string {
	return c.collectorImage
}

// CollectorConfigMapEntry represents the configuration file name for the collector. Immutable.
func (c *Config) CollectorConfigMapEntry() string {
	return c.collectorConfigMapEntry
}

// TargetAllocatorImage represents the flag to override the OpenTelemetry TargetAllocator container image.
func (c *Config) TargetAllocatorImage() string {
	return c.targetAllocatorImage
}

// TargetAllocatorConfigMapEntry represents the configuration file name for the TargetAllocator. Immutable.
func (c *Config) TargetAllocatorConfigMapEntry() string {
	return c.targetAllocatorConfigMapEntry
}

// OpenShiftRoutes represents the availability of the OpenShift Routes API.
func (c *Config) OpenShiftRoutes() autodetect.OpenShiftRoutesAvailability {
	return c.openshiftRoutes.Get()
}

// AutoInstrumentationJavaImage returns OpenTelemetry Java auto-instrumentation container image.
func (c *Config) AutoInstrumentationJavaImage() string {
	return c.autoInstrumentationJavaImage
}

// AutoInstrumentationNodeJSImage returns OpenTelemetry NodeJS auto-instrumentation container image.
func (c *Config) AutoInstrumentationNodeJSImage() string {
	return c.autoInstrumentationNodeJSImage
}

// AutoInstrumentationPythonImage returns OpenTelemetry Python auto-instrumentation container image.
func (c *Config) AutoInstrumentationPythonImage() string {
	return c.autoInstrumentationPythonImage
}

// AutoInstrumentationDotNetImage returns OpenTelemetry DotNet auto-instrumentation container image.
func (c *Config) AutoInstrumentationDotNetImage() string {
	return c.autoInstrumentationDotNetImage
}

// AutoInstrumentationGoImage returns OpenTelemetry Go auto-instrumentation container image.
func (c *Config) AutoInstrumentationGoImage() string {
	return c.autoInstrumentationGoImage
}

// AutoInstrumentationApacheHttpdImage returns OpenTelemetry ApacheHttpd auto-instrumentation container image.
func (c *Config) AutoInstrumentationApacheHttpdImage() string {
	return c.autoInstrumentationApacheHttpdImage
}

// AutoInstrumentationNginxImage returns OpenTelemetry Nginx auto-instrumentation container image.
func (c *Config) AutoInstrumentationNginxImage() string {
	return c.autoInstrumentationNginxImage
}

// LabelsFilter Returns the filters converted to regex strings used to filter out unwanted labels from propagations.
func (c *Config) LabelsFilter() []string {
	return c.labelsFilter
}

// RegisterOpenShiftRoutesChangeCallback registers the given function as a callback that
// is called when the OpenShift Routes detection detects a change.
func (c *Config) RegisterOpenShiftRoutesChangeCallback(f func() error) {
	c.onOpenShiftRoutesChange.Register(f)
}

type openshiftRoutesStore interface {
	Set(ora autodetect.OpenShiftRoutesAvailability)
	Get() autodetect.OpenShiftRoutesAvailability
}

func newOpenShiftRoutesWrapper() openshiftRoutesStore {
	return &openshiftRoutesWrapper{
		current: autodetect.OpenShiftRoutesNotAvailable,
	}
}

type openshiftRoutesWrapper struct {
	mu      sync.Mutex
	current autodetect.OpenShiftRoutesAvailability
}

func (p *openshiftRoutesWrapper) Set(ora autodetect.OpenShiftRoutesAvailability) {
	p.mu.Lock()
	p.current = ora
	p.mu.Unlock()
}

func (p *openshiftRoutesWrapper) Get() autodetect.OpenShiftRoutesAvailability {
	p.mu.Lock()
	ora := p.current
	p.mu.Unlock()
	return ora
}
