package main

import (
	"fmt"
	"strconv"

	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu-community/sensu-plugin-sdk/templates"
	"github.com/sensu/sensu-go/types"
)

// Config represents the mutator plugin config.
type Config struct {
	sensu.PluginConfig
	MetricNameTemplate string
}

var (
	mutatorConfig = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-influxdb-metric-mutator",
			Short:    "Sensu InfluxDB Metric Mutator",
			Keyspace: "sensu.io/plugins/sensu-influxdb-metric-mutator/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		{
			Path:      "metric-name-template",
			Env:       "METRIC_NAME_TEMPLATE",
			Argument:  "metric-name-template",
			Shorthand: "t",
			Default:   "{{.Check.Name}}.status",
			Usage:     "Template for naming the metric point for the check status",
			Value:     &mutatorConfig.MetricNameTemplate,
		},
	}
)

func main() {
	mutator := sensu.NewGoMutator(&mutatorConfig.PluginConfig, options, checkArgs, executeMutator)
	mutator.Execute()
}

func checkArgs(_ *types.Event) error {
	if len(mutatorConfig.MetricNameTemplate) == 0 {
		return fmt.Errorf("--MetricNameTemplate or METRIC_NAME_TEMPLATE environment variable is required")
	}
	return nil
}

func ternaryFunction(first string, second string) string {
	if first != "" {
		return first
	} else {
		return second
	}
}

func executeMutator(event *types.Event) (*types.Event, error) {
	if !event.HasCheck() {
		return &types.Event{}, fmt.Errorf("Event does not have a check defined.")
	}
	metricName, err := templates.EvalTemplate("metricName", mutatorConfig.MetricNameTemplate, event)
	if err != nil {
		return &types.Event{}, fmt.Errorf("Failed to evalutate template: %v", err)
	}

	// Possible TODO:  replace any spaces periods from the templated metricName

	// This really shouldn't happen if a metrics handler is defined, but just in case.
	if !event.HasMetrics() {
		event.Metrics = new(types.Metrics)
	}

	// Provide some extra information in the tags
	mt := make([]*types.MetricTag, 0)
	mt = append(mt, &types.MetricTag{Name: "critical", Value: event.Check.Labels["critical"]})
	mt = append(mt, &types.MetricTag{Name: "deployment_uid", Value: "none"})
	mt = append(mt, &types.MetricTag{Name: "host", Value: event.Entity.System.Hostname})
	mt = append(mt, &types.MetricTag{Name: "metric_source", Value: "sensu"})
	mt = append(mt, &types.MetricTag{Name: "name", Value: ternaryFunction(event.Check.Labels["display_name"], event.Check.Name)})
	mt = append(mt, &types.MetricTag{Name: "product_id", Value: event.Check.Labels["product"]})
	mt = append(mt, &types.MetricTag{Name: "ret_code", Value: strconv.FormatInt(int64(event.Check.Status), 10)})
	mt = append(mt, &types.MetricTag{Name: "status", Value: strconv.FormatInt(int64(event.Check.Status), 10)})
	mt = append(mt, &types.MetricTag{Name: "target_alias", Value: event.Check.Labels["service"]})
	mt = append(mt, &types.MetricTag{Name: "type", Value: "sensu"})
	mt = append(mt, &types.MetricTag{Name: "subproduct", Value: ternaryFunction(event.Check.Labels["subproduct"], event.Check.Labels["product"])})
	mt = append(mt, &types.MetricTag{Name: "app_name", Value: ternaryFunction(event.Check.Labels["app_name"], event.Check.Labels["service"])})

	mp := &types.MetricPoint{
		Name:      metricName,
		Value:     float64(event.Check.Status),
		Timestamp: event.Timestamp,
		Tags:      mt,
	}
	event.Metrics.Points = append(event.Metrics.Points, mp)
	return event, nil
}
