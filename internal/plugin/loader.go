package plugin

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/component"
)

// Loader handles loading and configuring plugins
type Loader struct {
	registry *Registry
	logger   hclog.Logger
}

// NewLoader creates a new plugin loader
func NewLoader(registry *Registry, logger hclog.Logger) *Loader {
	if registry == nil {
		registry = globalRegistry
	}
	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	return &Loader{
		registry: registry,
		logger:   logger,
	}
}

// LoadBuilder loads and configures a builder plugin
func (l *Loader) LoadBuilder(name string, config map[string]interface{}) (component.Builder, error) {
	l.logger.Debug("loading builder plugin", "name", name)

	builder, err := l.registry.GetBuilder(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get builder plugin %q: %w", name, err)
	}

	// Create a new instance if the builder is a pointer type
	builder = cloneComponent(builder).(component.Builder)

	// Configure the builder
	if err := configureComponent(builder, config); err != nil {
		return nil, fmt.Errorf("failed to configure builder %q: %w", name, err)
	}

	l.logger.Debug("builder plugin loaded", "name", name)
	return builder, nil
}

// LoadRegistry loads and configures a registry plugin
func (l *Loader) LoadRegistry(name string, config map[string]interface{}) (component.Registry, error) {
	l.logger.Debug("loading registry plugin", "name", name)

	registry, err := l.registry.GetRegistry(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry plugin %q: %w", name, err)
	}

	// Create a new instance
	registry = cloneComponent(registry).(component.Registry)

	// Configure the registry
	if err := configureComponent(registry, config); err != nil {
		return nil, fmt.Errorf("failed to configure registry %q: %w", name, err)
	}

	l.logger.Debug("registry plugin loaded", "name", name)
	return registry, nil
}

// LoadPlatform loads and configures a platform plugin
func (l *Loader) LoadPlatform(name string, config map[string]interface{}) (component.Platform, error) {
	l.logger.Debug("loading platform plugin", "name", name)

	platform, err := l.registry.GetPlatform(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get platform plugin %q: %w", name, err)
	}

	// Create a new instance
	platform = cloneComponent(platform).(component.Platform)

	// Configure the platform
	if err := configureComponent(platform, config); err != nil {
		return nil, fmt.Errorf("failed to configure platform %q: %w", name, err)
	}

	l.logger.Debug("platform plugin loaded", "name", name)
	return platform, nil
}

// LoadReleaseManager loads and configures a release manager plugin
func (l *Loader) LoadReleaseManager(name string, config map[string]interface{}) (component.ReleaseManager, error) {
	l.logger.Debug("loading release manager plugin", "name", name)

	rm, err := l.registry.GetReleaseManager(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get release manager plugin %q: %w", name, err)
	}

	// Create a new instance
	rm = cloneComponent(rm).(component.ReleaseManager)

	// Configure the release manager
	if err := configureComponent(rm, config); err != nil {
		return nil, fmt.Errorf("failed to configure release manager %q: %w", name, err)
	}

	l.logger.Debug("release manager plugin loaded", "name", name)
	return rm, nil
}

// configureComponent configures a component with the given configuration
func configureComponent(comp component.Configurable, config map[string]interface{}) error {
	// Always call ConfigSet to allow plugins to initialize defaults and read env vars
	// Even with empty config, plugins may need to set up defaults or environment fallbacks
	if err := comp.ConfigSet(config); err != nil {
		return fmt.Errorf("component configuration failed: %w", err)
	}

	return nil
}

// cloneComponent creates a new instance of a component
// This is a simplified version - in production, you might want to use a factory pattern
func cloneComponent(comp interface{}) interface{} {
	if comp == nil {
		return nil
	}

	// Get the type of the component
	t := reflect.TypeOf(comp)

	// If it's a pointer, get the element type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Create a new instance
	newComp := reflect.New(t).Interface()

	return newComp
}

// ConfigureFromMap is a helper function to set struct fields from a map
// This is a basic implementation - you might want to use a library like mapstructure
func ConfigureFromMap(target interface{}, config map[string]interface{}) error {
	if config == nil || len(config) == 0 {
		return nil
	}

	// Use reflection to set fields
	v := reflect.ValueOf(target)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a struct or pointer to struct")
	}

	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get the field name (could also use tags)
		fieldName := fieldType.Name

		// Look for the value in the config map
		if val, ok := config[fieldName]; ok {
			// Try to set the field value
			if err := setFieldValue(field, val); err != nil {
				return fmt.Errorf("failed to set field %s: %w", fieldName, err)
			}
		}
	}

	return nil
}

// setFieldValue sets a reflect.Value from an interface{} value
func setFieldValue(field reflect.Value, value interface{}) error {
	if value == nil {
		return nil
	}

	val := reflect.ValueOf(value)

	// Check if types are compatible
	if !val.Type().AssignableTo(field.Type()) {
		// Try to convert
		if val.Type().ConvertibleTo(field.Type()) {
			val = val.Convert(field.Type())
		} else {
			return fmt.Errorf("cannot assign %s to %s", val.Type(), field.Type())
		}
	}

	field.Set(val)
	return nil
}
