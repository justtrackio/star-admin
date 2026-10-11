package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/gosoline-project/sqlc"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

func TestStarRocksConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name     string
		override string
		host     string
	}{
		{name: "default", host: "starrocks.dev.example.com"},
		{name: "sandbox", override: "app:\n  env: sandbox\n", host: "starrocks.dev.example.com"},
		{name: "prod", override: "app:\n  env: prod\nsqlc:\n  default:\n    uri:\n      host: starrocks.prod.example.com\n", host: "starrocks.prod.example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := readStarRocksTestConfig(t, tc.override)
			if config.IsSet("sqlc.prod") || config.IsSet("sqlc.sdlc") || config.IsSet("starrocks.cluster") {
				t.Fatal("expected a single default connection without a cluster selector")
			}

			var err error
			var settings *sqlc.Settings

			if settings, err = sqlc.ReadSettings(config, "default"); err != nil {
				t.Fatal(err)
			}

			expectedUri := sqlc.SettingsUri{Host: tc.host, Port: 9030, User: "root"}
			if settings.Driver != sqlc.DriverMysql || settings.Uri != expectedUri {
				t.Fatalf("unexpected connection settings: %+v", settings)
			}

			if settings.Migrations.Enabled || settings.MultiStatements {
				t.Fatal("existing cluster must not enable migrations or multi-statements")
			}

			assertResourceLifecyclesDisabled(t, config)
		})
	}
}

func readStarRocksTestConfig(t *testing.T, override string) cfg.Config {
	t.Helper()
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	options := []cfg.Option{
		cfg.WithConfigFile("../config.dist.yml", "yml"),
		cfg.WithConfigBytes([]byte(`sqlc:
  default:
    uri:
      host: starrocks.dev.example.com
resource_lifecycles:
  create:
    enabled: false
  init:
    enabled: false
  register:
    enabled: false
  purge:
    enabled: false
`), "yml"),
	}

	if override != "" {
		options = append(options, cfg.WithConfigBytes([]byte(override), "yml"))
	}

	if err := config.Option(options...); err != nil {
		t.Fatal(err)
	}

	return config
}

func assertResourceLifecyclesDisabled(t *testing.T, config cfg.Config) {
	t.Helper()
	for _, cycle := range []string{"create", "init", "register", "purge"} {
		enabled, err := config.GetBool("resource_lifecycles." + cycle + ".enabled")
		if err != nil || enabled {
			t.Fatalf("resource lifecycle %s: enabled=%v, error=%v", cycle, enabled, err)
		}
	}
}

type closeTrackingClient struct {
	sqlc.Client
	calls int
	err   error
}

func (c *closeTrackingClient) Close() error {
	c.calls++

	return c.err
}

func TestStarRocksPoolClosesOnShutdown(t *testing.T) {
	closeError := errors.New("close failed")
	for _, expected := range []error{nil, closeError} {
		ctx, cancel := context.WithCancel(context.Background())
		client := &closeTrackingClient{err: expected}
		module := &ModuleStarRocks{client: client}
		cancel()

		err := module.Run(ctx)
		if !errors.Is(err, expected) {
			t.Fatalf("shutdown error = %v, want %v", err, expected)
		}

		if client.calls != 1 {
			t.Fatalf("pool closed %d times, want once", client.calls)
		}
	}
}
