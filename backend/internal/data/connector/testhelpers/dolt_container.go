package testhelpers

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/clario360/platform/internal/data/model"
)

const (
	doltImage        = "dolthub/dolt-sql-server:latest"
	doltDatabaseName = "app"
	doltUser         = "clario"
	doltPassword     = "clariopass"
	doltRootPassword = "rootpass"
)

type DoltContainer struct {
	Container    tc.Container
	Host         string
	Port         int
	Database     string
	Username     string
	Password     string
	RootPassword string
}

func StartDoltContainer(ctx context.Context, t testing.TB, database string) *DoltContainer {
	t.Helper()
	SkipIfDockerUnavailable(t)
	if database == "" {
		database = doltDatabaseName
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image:        doltImage,
			ExposedPorts: []string{"3306/tcp"},
			Env: map[string]string{
				"DOLT_ROOT_PASSWORD": doltRootPassword,
				"DOLT_ROOT_HOST":     "%",
				"DOLT_DATABASE":      database,
				"DOLT_USER":          doltUser,
				"DOLT_PASSWORD":      doltPassword,
				"DOLT_USER_HOST":     "%",
			},
			WaitingFor: wait.ForLog("Ready for connections.").
				WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start dolt container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("dolt host: %v", err)
	}
	port, err := container.MappedPort(ctx, "3306/tcp")
	if err != nil {
		t.Fatalf("dolt port: %v", err)
	}

	return &DoltContainer{
		Container:    container,
		Host:         host,
		Port:         port.Int(),
		Database:     database,
		Username:     doltUser,
		Password:     doltPassword,
		RootPassword: doltRootPassword,
	}
}

func (c *DoltContainer) ConnectionConfig(branch string) model.DoltConnectionConfig {
	if branch == "" {
		branch = "main"
	}
	return model.DoltConnectionConfig{
		Host:                c.Host,
		Port:                c.Port,
		Database:            c.Database,
		Username:            c.Username,
		Password:            c.Password,
		Branch:              branch,
		MaxOpenConns:        1,
		MaxIdleConns:        1,
		ConnMaxLifetimeMins: 30,
	}
}

func (c *DoltContainer) OpenDB(ctx context.Context, username, password string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", username, password, c.Host, c.Port, c.Database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (c *DoltContainer) OpenUserDB(ctx context.Context) (*sql.DB, error) {
	return c.OpenDB(ctx, c.Username, c.Password)
}

func (c *DoltContainer) OpenRootDB(ctx context.Context) (*sql.DB, error) {
	return c.OpenDB(ctx, "root", c.RootPassword)
}

func (c *DoltContainer) ExecRepoShell(ctx context.Context, shell string) (string, error) {
	cmd := []string{"sh", "-lc", fmt.Sprintf("cd /var/lib/dolt/%s && %s", c.Database, shell)}
	code, reader, err := c.Container.Exec(ctx, cmd)
	if err != nil {
		return "", err
	}
	body, readErr := io.ReadAll(reader)
	if readErr != nil {
		return "", readErr
	}
	if code != 0 {
		return string(body), fmt.Errorf("dolt exec failed with exit code %d: %s", code, strings.TrimSpace(string(body)))
	}
	return string(body), nil
}
