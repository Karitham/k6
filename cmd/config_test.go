/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2019 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package cmd

import (
	"testing"
	"time"

	"github.com/mstoykov/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.k6.io/k6/errext"
	"go.k6.io/k6/errext/exitcodes"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/lib/executor"
	"go.k6.io/k6/lib/metrics"
	"go.k6.io/k6/lib/testutils"
	"go.k6.io/k6/lib/types"
	"go.k6.io/k6/stats"
	"gopkg.in/guregu/null.v3"
)

type testCmdData struct {
	Name  string
	Tests []testCmdTest
}

type testCmdTest struct {
	Args     []string
	Expected []string
	Name     string
}

func TestConfigCmd(t *testing.T) {
	t.Parallel()
	testdata := []testCmdData{
		{
			Name: "Out",

			Tests: []testCmdTest{
				{
					Name:     "NoArgs",
					Args:     []string{""},
					Expected: []string{},
				},
				{
					Name:     "SingleArg",
					Args:     []string{"--out", "influxdb=http://localhost:8086/k6"},
					Expected: []string{"influxdb=http://localhost:8086/k6"},
				},
				{
					Name:     "MultiArg",
					Args:     []string{"--out", "influxdb=http://localhost:8086/k6", "--out", "json=test.json"},
					Expected: []string{"influxdb=http://localhost:8086/k6", "json=test.json"},
				},
			},
		},
	}

	for _, data := range testdata {
		t.Run(data.Name, func(t *testing.T) {
			t.Parallel()
			for _, test := range data.Tests {
				t.Run(`"`+test.Name+`"`, func(t *testing.T) {
					t.Parallel()
					fs := configFlagSet()
					fs.AddFlagSet(optionFlagSet())
					assert.NoError(t, fs.Parse(test.Args))

					config, err := getConfig(fs)
					assert.NoError(t, err)
					assert.Equal(t, test.Expected, config.Out)
				})
			}
		})
	}
}

func TestConfigEnv(t *testing.T) {
	t.Parallel()
	testdata := map[struct{ Name, Key string }]map[string]func(Config){
		{"Linger", "K6_LINGER"}: {
			"":      func(c Config) { assert.Equal(t, null.Bool{}, c.Linger) },
			"true":  func(c Config) { assert.Equal(t, null.BoolFrom(true), c.Linger) },
			"false": func(c Config) { assert.Equal(t, null.BoolFrom(false), c.Linger) },
		},
		{"NoUsageReport", "K6_NO_USAGE_REPORT"}: {
			"":      func(c Config) { assert.Equal(t, null.Bool{}, c.NoUsageReport) },
			"true":  func(c Config) { assert.Equal(t, null.BoolFrom(true), c.NoUsageReport) },
			"false": func(c Config) { assert.Equal(t, null.BoolFrom(false), c.NoUsageReport) },
		},
		{"Out", "K6_OUT"}: {
			"":         func(c Config) { assert.Equal(t, []string{}, c.Out) },
			"influxdb": func(c Config) { assert.Equal(t, []string{"influxdb"}, c.Out) },
		},
	}
	for field, data := range testdata {
		field, data := field, data
		t.Run(field.Name, func(t *testing.T) {
			t.Parallel()
			for value, fn := range data {
				value, fn := value, fn
				t.Run(`"`+value+`"`, func(t *testing.T) {
					t.Parallel()
					var config Config
					assert.NoError(t, envconfig.Process("", &config, func(key string) (string, bool) {
						if key == field.Key {
							return value, true
						}
						return "", false
					}))
					fn(config)
				})
			}
		})
	}
}

func TestConfigApply(t *testing.T) {
	t.Parallel()
	t.Run("Linger", func(t *testing.T) {
		t.Parallel()
		conf := Config{}.Apply(Config{Linger: null.BoolFrom(true)})
		assert.Equal(t, null.BoolFrom(true), conf.Linger)
	})
	t.Run("NoUsageReport", func(t *testing.T) {
		t.Parallel()
		conf := Config{}.Apply(Config{NoUsageReport: null.BoolFrom(true)})
		assert.Equal(t, null.BoolFrom(true), conf.NoUsageReport)
	})
	t.Run("Out", func(t *testing.T) {
		t.Parallel()
		conf := Config{}.Apply(Config{Out: []string{"influxdb"}})
		assert.Equal(t, []string{"influxdb"}, conf.Out)

		conf = Config{}.Apply(Config{Out: []string{"influxdb", "json"}})
		assert.Equal(t, []string{"influxdb", "json"}, conf.Out)
	})
}

func TestDeriveAndValidateConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		conf   Config
		isExec bool
		err    string
	}{
		{"defaultOK", Config{}, true, ""},
		{
			"defaultErr",
			Config{},
			false,
			"executor default: function 'default' not found in exports",
		},
		{
			"nonDefaultOK", Config{Options: lib.Options{Scenarios: lib.ScenarioConfigs{
				"per_vu_iters": executor.PerVUIterationsConfig{
					BaseConfig: executor.BaseConfig{
						Name: "per_vu_iters", Type: "per-vu-iterations", Exec: null.StringFrom("nonDefault"),
					},
					VUs:         null.IntFrom(1),
					Iterations:  null.IntFrom(1),
					MaxDuration: types.NullDurationFrom(time.Second),
				},
			}}}, true, "",
		},
		{
			"nonDefaultErr",
			Config{Options: lib.Options{Scenarios: lib.ScenarioConfigs{
				"per_vu_iters": executor.PerVUIterationsConfig{
					BaseConfig: executor.BaseConfig{
						Name: "per_vu_iters", Type: "per-vu-iterations", Exec: null.StringFrom("nonDefaultErr"),
					},
					VUs:         null.IntFrom(1),
					Iterations:  null.IntFrom(1),
					MaxDuration: types.NullDurationFrom(time.Second),
				},
			}}},
			false,
			"executor per_vu_iters: function 'nonDefaultErr' not found in exports",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := deriveAndValidateConfig(tc.conf,
				func(_ string) bool { return tc.isExec })
			if tc.err != "" {
				var ecerr errext.HasExitCode
				assert.ErrorAs(t, err, &ecerr)
				assert.Equal(t, exitcodes.InvalidConfig, ecerr.ExitCode())
				assert.Contains(t, err.Error(), tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateThresholdsConfigWithNilRegistry(t *testing.T) {
	var registry *metrics.Registry
	config := Config{}
	var wantErrType errext.HasExitCode

	gotErr := validateThresholdsConfig(config, registry)

	assert.Error(t, gotErr, "validateThresholdsConfig should fail when passed registry is nil")
	assert.ErrorAs(t, gotErr, &wantErrType, "validateThresholdsConfig error should be an instance of errext.HasExitCode")
}

func TestValidateThresholdsConfigAppliesToBuiltinMetrics(t *testing.T) {
	// Prepare a registry loaded with builtin metrics
	registry := metrics.NewRegistry()
	metrics.RegisterBuiltinMetrics(registry)

	// Assuming builtin metrics are indeed registered, and
	// thresholds parsing works as expected, we prepare
	// thresholds for a counter builting metric; namely http_reqs
	HTTPReqsThresholds, err := stats.NewThresholds([]string{"count>0", "rate>1"})
	require.NoError(t, err, "instantiating Thresholds with expression 'count>0' should not fail")
	options := lib.Options{
		Thresholds: map[string]stats.Thresholds{
			metrics.HTTPReqsName: HTTPReqsThresholds,
		},
	}
	config := Config{Options: options}

	gotErr := validateThresholdsConfig(config, registry)

	assert.NoError(t, gotErr, "validateThresholdsConfig should not fail against builtin metrics")
}

func TestValidateThresholdsConfigAppliesToCustomMetrics(t *testing.T) {
	t.Parallel()

	// Prepare a registry loaded with both builtin metrics,
	// and a custom counter metric.
	testCounterMetricName := "testcounter"
	registry := metrics.NewRegistry()
	metrics.RegisterBuiltinMetrics(registry)
	_, err := registry.NewMetric(testCounterMetricName, stats.Counter)
	require.NoError(t, err, "registering custom counter metric should not fail")

	// Prepare a configuration containing a Threshold
	counterThresholds, err := stats.NewThresholds([]string{"count>0", "rate>1"})
	require.NoError(t, err, "instantiating Thresholds with expression 'count>0' should not fail")
	options := lib.Options{
		Thresholds: map[string]stats.Thresholds{
			testCounterMetricName: counterThresholds,
		},
	}
	config := Config{Options: options}

	gotErr := validateThresholdsConfig(config, registry)

	// Assert
	assert.NoError(t, gotErr, "validateThresholdsConfig should not fail against existing and valid custom metric")
}

func TestValidateThresholdsConfigFailsOnNonExistingMetric(t *testing.T) {
	t.Parallel()

	// Prepare a registry loaded with builtin metrics only
	registry := metrics.NewRegistry()
	metrics.RegisterBuiltinMetrics(registry)

	// Prepare a configuration containing a Threshold applying to
	// a non-existing metric
	counterThresholds, err := stats.NewThresholds([]string{"count>0", "rate>1"})
	require.NoError(t, err, "instantiating Thresholds with expression 'count>0' should not fail")
	options := lib.Options{
		Thresholds: map[string]stats.Thresholds{
			"nonexisting": counterThresholds,
		},
	}
	config := Config{Options: options}
	var wantErrType errext.HasExitCode

	gotErr := validateThresholdsConfig(config, registry)

	// Assert
	assert.Error(t, gotErr, "validateThresholdsConfig should fail on thresholds applied to a non-existing metric")
	assert.ErrorAs(t, gotErr, &wantErrType, "validateThresholdsConfig error should be an instance of errext.HasExitCode")
}

func TestValidateThresholdsConfigFailsOnThresholdInvalidMetricType(t *testing.T) {
	t.Parallel()

	// Prepare a registry loaded with builtin metrics only
	registry := metrics.NewRegistry()
	metrics.RegisterBuiltinMetrics(registry)

	// Prepare a configuration containing a Threshold using a Counter metric
	// specific aggregation method, against a metric of type Gauge: which doesn't support
	// that method.
	VUsThresholds, err := stats.NewThresholds([]string{"count>0"})
	require.NoError(t, err, "instantiating Thresholds with expression 'count>0' should not fail")
	options := lib.Options{
		Thresholds: map[string]stats.Thresholds{
			metrics.VUsName: VUsThresholds,
		},
	}
	config := Config{Options: options}
	var wantErrType errext.HasExitCode

	gotErr := validateThresholdsConfig(config, registry)

	// Assert
	assert.Error(t, gotErr, "validateThresholdsConfig should fail applying the count method to a Gauge metric")
	assert.ErrorAs(t, gotErr, &wantErrType, "validateThresholdsConfig error should be an instance of errext.HasExitCode")
}
