package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"k8s.io/utils/ptr"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/domains"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/registeredlimits"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/services"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

var (
	onStartFunc func(context.Context) error
	onStopFunc  func(context.Context) error
)

func main() {
	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "/log"
	}

	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	podName := os.Getenv("POD_NAME")
	if podName == "" {
		namespace = "default"
	}

	testCase := os.Getenv("TEST_CASE")
	if testCase == "" {
		testCase = "unknown-case"
	}

	logPath = path.Join(logPath, testCase, namespace)

	if err := os.MkdirAll(logPath, 0755); err != nil {
		panic(err)
	}

	f, err := os.OpenFile(path.Join(logPath, "log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	slog.SetDefault(slog.New(slog.NewTextHandler(io.MultiWriter(os.Stderr, f), nil)).With("podName", podName))

	slog.Info("starting")
	ctx, cancel := signal.NotifyContext(context.TODO(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	slog.Info("environment variables ", "env", os.Environ())

	var failed bool
	if onStartFunc != nil {
		slog.Info("running onStart...")
		if err := onStartFunc(ctx); err != nil {
			slog.Error("executing onStart", "error", err)
			failed = true
		}
	}

	if !failed {
		slog.Info("onStart completed")

		readinessFile := os.Getenv("READINESS_FILE")
		if readinessFile == "" {
			slog.Error("READINESS_FILE not found")
			os.Exit(1)
		}

		if err := os.Mkdir(path.Dir(readinessFile), 0755); err != nil {
			slog.Error("create path for readiness file", "readinessFile", readinessFile, "error", err)
			os.Exit(1)
		}

		if _, err := os.Create(readinessFile); err != nil {
			slog.Error("create readiness file", "readinessFile", readinessFile, "error", err)
			os.Exit(1)
		}
	}

	<-ctx.Done()

	slog.Info("signal received")

	ctx1, cancel1 := context.WithTimeout(context.TODO(), time.Second*30)
	defer cancel1()

	if onStopFunc != nil {
		slog.Info("running onStop...")
		if err := onStopFunc(ctx1); err != nil {
			slog.Error("executing onStop", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("onStop completed")

	slog.Info("exiting")
}

func newKeystoneClient(ctx context.Context) (*gophercloud.ServiceClient, error) {
	cloudName := os.Getenv("OS_CLOUD")
	if cloudName == "" {
		cloudName = "openstack-admin"
	}

	opts := &clientconfig.ClientOpts{
		Cloud: cloudName,
	}

	p, err := clientconfig.AuthenticatedClient(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("get authenticated client: %w", err)
	}

	return openstack.NewIdentityV3(p, gophercloud.EndpointOpts{})
}

type errorDetail struct {
	Title   string `json:"title,omitempty"`
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type errorDetailWrapper struct {
	Error errorDetail `json:"error"`
}

func extractErrorDetail(err error) (*errorDetail, bool) {
	gcErr := &gophercloud.ErrUnexpectedResponseCode{}
	if !errors.As(err, gcErr) {
		return nil, false
	}
	detail := &errorDetailWrapper{}
	if err := json.Unmarshal(gcErr.Body, detail); err != nil {
		return nil, false
	}

	return &detail.Error, true
}

func retry(ctx context.Context, delay time.Duration, f func(ctx context.Context) (bool, error)) error {
	for {
		ok, err := f(ctx)
		if err != nil {
			return err
		}

		if !ok {
			return nil
		}

		time.Sleep(delay)
	}
}

func deleteRegisteredLimit(ctx context.Context, c *gophercloud.ServiceClient, id string) error {
	slog := slog.With("id", id)
	return retry(ctx, time.Second, func(ctx context.Context) (bool, error) {
		if err := registeredlimits.Delete(ctx, c, id).ExtractErr(); err != nil {
			detail, ok := extractErrorDetail(err)
			if !ok {
				return false, err
			}

			if detail.Code == 404 {
				slog.Error("registered limit has been deleted")
				return false, nil
			}
			if detail.Code == 403 && strings.Contains(detail.Message, "because there are project limits associated with it") {
				slog.Info("there are project limits associated with it")
				return true, nil
			}

			return false, err
		}

		return false, nil
	})
}

func disableDomain(ctx context.Context, c *gophercloud.ServiceClient, domainName string) {
	allPages, err := domains.List(c, domains.ListOpts{Name: domainName}).AllPages(ctx)
	if err != nil {
		slog.Error("find domain", "domainName", domainName, "error", err)
		return
	}

	dms, err := domains.ExtractDomains(allPages)
	if err != nil {
		slog.Error("extract domain result", "error", err)
		return
	}

	for _, d := range dms {
		if d.Enabled {
			slog := slog.With("domainName", d.Name, "domainId", d.ID)
			if _, err := domains.Update(ctx, c, d.ID, domains.UpdateOpts{Enabled: ptr.To(false)}).Extract(); err != nil {
				slog.Error("set domain to disabled", "error", err)
			} else {
				slog.Info("disabled domain")
			}
		}
	}
}

func getServiceId(ctx context.Context, c *gophercloud.ServiceClient, svcName string) (string, error) {
	var svc *services.Service

	if err := retry(ctx, time.Second, func(ctx context.Context) (bool, error) {
		allPages, err := services.List(c, services.ListOpts{Name: svcName}).AllPages(ctx)
		if err != nil {
			return false, fmt.Errorf("list service: %w", err)
		}
		svcs, err := services.ExtractServices(allPages)
		if err != nil {
			return false, fmt.Errorf("extract services: %w", err)
		}

		switch len(svcs) {
		case 0:
			slog.Info("no service found", "serviceName", svcName)
			return true, nil
		case 1:
			svc = &svcs[0]
			return false, nil
		default:
			return false, fmt.Errorf("more than one service found")
		}
	}); err != nil {
		return "", err
	}

	return svc.ID, nil
}
