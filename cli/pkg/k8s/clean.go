package k8s

import (
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/cli/common"
	"github.com/EPOS-ERIC/epos-platform/cli/display"
	"github.com/EPOS-ERIC/epos-platform/cli/validate"
)

// CleanOpts defines inputs for Clean.
type CleanOpts struct {
	// Required. name of the environment
	Name string
	// Optional. Kubernetes context to use; defaults to the current kubectl context when unset.
	Context string
	// Optional. Timeout for each clean job execution; defaults when unset.
	Timeout time.Duration
}

// Clean is not yet implemented for the current Kubernetes stack.
func Clean(opts CleanOpts) (*Env, error) {
	// TODO: Implement cleanup for the current stack while preserving AAI data.
	return nil, fmt.Errorf("kubernetes clean is not implemented")
}

func getRenderedTemplateBySuffix(files map[string]string, suffix string) (string, error) {
	for name, content := range files {
		if !strings.HasSuffix(name, suffix) {
			continue
		}

		trimmed := strings.TrimSpace(content)
		if trimmed == "" || trimmed == "---" {
			return "", fmt.Errorf("template %s rendered empty output", name)
		}

		return content, nil
	}

	return "", fmt.Errorf("template with suffix %s not found in rendered output", suffix)
}

func runOneOffJobFromManifest(context, namespace, jobName, manifest string, timeout time.Duration) error {
	display.Step("Running clean job: %s", jobName)

	if err := runKubectl(".", true, context, "delete", "job", jobName, "-n", namespace, "--ignore-not-found=true"); err != nil {
		return fmt.Errorf("failed to delete existing job %s before rerun: %w", jobName, err)
	}

	if err := runKubectlWithInput(".", false, context, manifest, "apply", "-f", "-", "-n", namespace); err != nil {
		return fmt.Errorf("failed to apply job manifest for %s: %w", jobName, err)
	}

	timeoutSeconds := int(timeout.Seconds())
	if err := runKubectl(
		".",
		false,
		context,
		"wait",
		"--for=condition=complete",
		fmt.Sprintf("job/%s", jobName),
		"-n",
		namespace,
		"--timeout",
		fmt.Sprintf("%ds", timeoutSeconds),
	); err != nil {
		describe := runKubectlCapture(".", context, "describe", "job", jobName, "-n", namespace)
		logs := runKubectlCapture(".", context, "logs", fmt.Sprintf("job/%s", jobName), "-n", namespace, "--all-containers=true")

		parts := []string{}
		if describe != "" {
			parts = append(parts, "kubectl describe output:\n"+describe)
		}
		if logs != "" {
			parts = append(parts, "kubectl logs output:\n"+logs)
		}

		diagnostics := strings.Join(parts, "\n\n")
		if diagnostics != "" {
			return fmt.Errorf("job %s did not complete successfully: %w\n%s", jobName, err, diagnostics)
		}
		return fmt.Errorf("job %s did not complete successfully: %w", jobName, err)
	}

	display.Done("Completed clean job: %s", jobName)

	return nil
}

// Validate checks CleanOpts and ensures the target environment is valid and reachable.
func (c *CleanOpts) Validate() error {
	display.Debug("name: %s", c.Name)
	display.Debug("context: %s", c.Context)
	display.Debug("timeout: %s", c.Timeout)

	if c.Name == "" {
		return fmt.Errorf("environment name is required")
	}

	resolvedTimeout, err := resolveCommandTimeout(c.Timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout: %w", err)
	}

	c.Timeout = resolvedTimeout

	if err := validate.Name(c.Name); err != nil {
		return fmt.Errorf("'%s' is an invalid name for an environment: %w", c.Name, err)
	}

	if c.Context == "" {
		context, err := common.GetCurrentKubeContext()
		if err != nil {
			return fmt.Errorf("failed to get current kubectl context: %w", err)
		}

		c.Context = context

		display.Debug("using current kubectl context: %s", c.Context)
	} else if err := EnsureContextExists(c.Context); err != nil {
		return fmt.Errorf("K8s context %q is not an available context: %w", c.Context, err)
	}

	if err := EnsureEnvironmentExists(c.Name, c.Context); err != nil {
		return fmt.Errorf("no environment with the name '%s' exists: %w", c.Name, err)
	}

	return nil
}
