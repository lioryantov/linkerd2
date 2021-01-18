package api

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/linkerd/linkerd2/pkg/healthcheck"
	"github.com/linkerd/linkerd2/pkg/k8s"
	"github.com/linkerd/linkerd2/viz/metrics-api/client"
	pb "github.com/linkerd/linkerd2/viz/metrics-api/gen/viz"
	vizHealthCheck "github.com/linkerd/linkerd2/viz/pkg/healthcheck"
)

// RawClient creates a raw viz API client with no validation.
func RawClient(ctx context.Context, kubeAPI *k8s.KubernetesAPI, namespace string) (pb.ApiClient, error) {
	return client.NewExternalClient(ctx, namespace, kubeAPI)
}

// CheckClientOrExit builds a new Viz API client and executes default status
// checks to determine if the client can successfully perform cli commands. If the
// checks fail, then CLI will print an error and exit.
func CheckClientOrExit(hcOptions healthcheck.Options) pb.ApiClient {
	hcOptions.RetryDeadline = time.Time{}
	return CheckClientOrRetryOrExit(hcOptions, false)
}

// CheckClientOrRetryOrExit builds a new Viz API client and executes status
// checks to determine if the client can successfully connect to the API. If the
// checks fail, then CLI will print an error and exit. If the hcOptions.retryDeadline
// param is specified, then the CLI will print a message to stderr and retry.
func CheckClientOrRetryOrExit(hcOptions healthcheck.Options, apiChecks bool) pb.ApiClient {
	checks := []healthcheck.CategoryID{
		healthcheck.KubernetesAPIChecks,
		vizHealthCheck.LinkerdVizExtensionCheck,
	}

	if apiChecks {
		checks = append(checks, healthcheck.LinkerdAPIChecks)
	}

	hc := vizHealthCheck.NewHealthChecker(checks, &hcOptions)

	hc.RunChecks(exitOnError)
	return hc.VizAPIClient()
}

func exitOnError(result *healthcheck.CheckResult) {
	if result.Retry {
		fmt.Fprintln(os.Stderr, "Waiting for control plane to become available")
		return
	}

	if result.Err != nil && !result.Warning {
		var msg string
		switch result.Category {
		case healthcheck.KubernetesAPIChecks:
			msg = "Cannot connect to Kubernetes"
		case vizHealthCheck.LinkerdVizExtensionCheck:
			msg = "Cannot connect to Linkerd Viz"
		}
		fmt.Fprintf(os.Stderr, "%s: %s\n", msg, result.Err)

		checkCmd := "linkerd viz check"
		fmt.Fprintf(os.Stderr, "Validate the install with: %s\n", checkCmd)

		os.Exit(1)
	}
}