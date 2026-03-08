package testhelpers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/clario360/platform/internal/data/model"
)

const (
	clickHouseImage    = "clickhouse/clickhouse-server:latest"
	clickHouseUser     = "clario"
	clickHousePassword = "clariopass"
)

type ClickHouseContainer struct {
	Container       tc.Container
	Host            string
	NativePort      int
	HTTPPort        int
	Database        string
	Username        string
	Password        string
	NativeAvailable bool
}

func StartClickHouseContainer(ctx context.Context, t testing.TB, database string) *ClickHouseContainer {
	t.Helper()
	SkipIfDockerUnavailable(t)
	if database == "" {
		database = "default"
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image:        clickHouseImage,
			ExposedPorts: []string{"9000/tcp", "8123/tcp"},
			Env: map[string]string{
				"CLICKHOUSE_DB":                        database,
				"CLICKHOUSE_USER":                      clickHouseUser,
				"CLICKHOUSE_PASSWORD":                  clickHousePassword,
				"CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT": "1",
			},
			WaitingFor: wait.ForHTTP("/ping").
				WithPort("8123/tcp").
				WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start clickhouse container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("clickhouse host: %v", err)
	}
	httpPort, err := container.MappedPort(ctx, "8123/tcp")
	if err != nil {
		t.Fatalf("clickhouse http port: %v", err)
	}
	nativePort, err := container.MappedPort(ctx, "9000/tcp")
	nativeAvailable := err == nil
	if !nativeAvailable {
		nativePort = httpPort
	}

	return &ClickHouseContainer{
		Container:       container,
		Host:            host,
		NativePort:      nativePort.Int(),
		HTTPPort:        httpPort.Int(),
		Database:        database,
		Username:        clickHouseUser,
		Password:        clickHousePassword,
		NativeAvailable: nativeAvailable,
	}
}

func (c *ClickHouseContainer) NativeConfig(database string) model.ClickHouseConnectionConfig {
	if database == "" {
		database = c.Database
	}
	return model.ClickHouseConnectionConfig{
		Host:               c.Host,
		Port:               c.NativePort,
		Database:           database,
		Username:           c.Username,
		Password:           c.Password,
		Protocol:           clickHouseProtocol(c.NativeAvailable),
		MaxOpenConns:       4,
		MaxIdleConns:       2,
		DialTimeoutSeconds: 10,
		ReadTimeoutSeconds: 30,
		Compression:        true,
	}
}

func (c *ClickHouseContainer) HTTPConfig(database string) model.ClickHouseConnectionConfig {
	cfg := c.NativeConfig(database)
	cfg.Protocol = "http"
	cfg.Port = c.HTTPPort
	return cfg
}

func (c *ClickHouseContainer) OpenDB(ctx context.Context, database string) (*sql.DB, error) {
	if database == "" {
		database = c.Database
	}
	protocol := clickhouse.Native
	if !c.NativeAvailable {
		protocol = clickhouse.HTTP
	}
	db := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", c.Host, c.NativePort)},
		Auth: clickhouse.Auth{
			Database: database,
			Username: c.Username,
			Password: c.Password,
		},
		Protocol:    protocol,
		DialTimeout: time.Second * 10,
		ReadTimeout: time.Second * 30,
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
	})
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func clickHouseProtocol(nativeAvailable bool) string {
	if nativeAvailable {
		return "native"
	}
	return "http"
}
