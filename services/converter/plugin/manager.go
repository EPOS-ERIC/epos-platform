// Package plugin manages local plugin installations and execution.
package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/converter/domain"
	"github.com/google/uuid"
)

// entrypoints maps each runtime to the file that must exist in an installation.
var entrypoints = map[domain.Runtime]string{
	domain.RuntimePython: "plugin.py",
	domain.RuntimeJava:   "plugin.jar",
	domain.RuntimeBinary: "plugin",
}

type Definition struct {
	ID         uuid.UUID
	Repository url.URL
	Branch     string
	Runtime    domain.Runtime
	Arguments  []string
}

type Status struct {
	Installed bool
	Commit    string
}

type installationStore interface {
	Installation(context.Context, uuid.UUID) (*domain.Installation, error)
	SetInstallation(context.Context, uuid.UUID, *string) error
}

type Manager struct {
	root    string
	max     int64
	timeout time.Duration
	store   installationStore
}

func New(root string, maxBytes int64, timeout time.Duration, store installationStore) (*Manager, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve plugin directory: %w", err)
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("create plugin directory: %w", err)
	}
	return &Manager{root: root, max: maxBytes, timeout: timeout, store: store}, nil
}

func (m *Manager) Status(ctx context.Context, id uuid.UUID) (Status, error) {
	current, _, err := m.current(ctx, id)
	if err != nil {
		return Status{}, err
	}
	status := Status{Installed: current != nil}
	if current != nil {
		status.Commit = current.Commit
	}
	return status, nil
}

// current returns the active installation and its directory.
func (m *Manager) current(ctx context.Context, id uuid.UUID) (*domain.Installation, string, error) {
	installed, err := m.store.Installation(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("get installation: %w", err)
	}
	if installed == nil {
		return nil, "", nil
	}
	dir := filepath.Join(m.root, id.String())
	if err := validateEntrypoint(dir, installed.Runtime); err != nil {
		return nil, "", nil
	}
	if installed.Runtime == domain.RuntimePython {
		if _, err := os.Stat(filepath.Join(dir, "venv", "bin", "python")); err != nil {
			return nil, "", nil
		}
	}
	return installed, dir, nil
}

func validateEntrypoint(dir string, runtime domain.Runtime) error {
	name, ok := entrypoints[runtime]
	if !ok {
		return fmt.Errorf("unsupported runtime %q", runtime)
	}
	info, err := os.Lstat(filepath.Join(dir, name))
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("plugin entrypoint %s is missing or not a regular file", name)
	}
	if runtime == domain.RuntimeBinary && info.Mode().Perm()&0o111 == 0 {
		return errors.New("plugin entrypoint is not executable")
	}
	return nil
}

func (m *Manager) Sync(ctx context.Context, definition Definition, force bool) error {
	return m.install(ctx, definition, force)
}

// install replaces the plugin directory unless the installed commit is current.
func (m *Manager) install(ctx context.Context, definition Definition, force bool) error {
	previous, _, previousErr := m.current(ctx, definition.ID)
	if previousErr != nil {
		return previousErr
	}
	if previous != nil && !force {
		head, err := remoteCommit(ctx, definition)
		if err != nil {
			return fmt.Errorf("check plugin branch: %w", err)
		}
		if head == previous.Commit {
			return nil
		}
	}

	if err := m.store.SetInstallation(ctx, definition.ID, nil); err != nil {
		return fmt.Errorf("clear installation: %w", err)
	}
	dir := filepath.Join(m.root, definition.ID.String())
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove previous installation: %w", err)
	}
	commit, err := m.prepare(ctx, definition, dir)
	if err != nil {
		return err
	}
	if err := m.store.SetInstallation(ctx, definition.ID, &commit); err != nil {
		return fmt.Errorf("record installation: %w", err)
	}
	return nil
}

// prepare clones, validates and sets up the plugin in dir.
func (m *Manager) prepare(ctx context.Context, definition Definition, dir string) (string, error) {
	err := run(ctx, "", "git", "clone", "--single-branch", "--branch", definition.Branch, "--", definition.Repository.String(), dir)
	if err != nil {
		return "", fmt.Errorf("clone plugin: %w", err)
	}
	out, err := output(ctx, dir, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("read plugin commit: %w", err)
	}
	commit := strings.TrimSpace(out)
	if err := validateEntrypoint(dir, definition.Runtime); err != nil {
		return "", err
	}
	if definition.Runtime == domain.RuntimePython {
		if err := setupPython(ctx, dir); err != nil {
			return "", err
		}
	}
	return commit, nil
}

func setupPython(ctx context.Context, dir string) error {
	if err := run(ctx, dir, "python3", "-m", "venv", "venv"); err != nil {
		return fmt.Errorf("create python environment: %w", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("check requirements: %w", err)
	}
	if err := run(ctx, dir, filepath.Join(dir, "venv", "bin", "pip"), "install", "-r", "requirements.txt"); err != nil {
		return fmt.Errorf("install python dependencies: %w", err)
	}
	return nil
}

// remoteCommit returns the commit the plugin's branch currently points to.
func remoteCommit(ctx context.Context, definition Definition) (string, error) {
	out, err := output(ctx, "", "git", "ls-remote", "--exit-code", "--heads", "--",
		definition.Repository.String(), "refs/heads/"+definition.Branch)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return "", errors.New("unexpected empty ls-remote output")
	}
	return fields[0], nil
}

func run(ctx context.Context, dir, name string, args ...string) error {
	_, err := output(ctx, dir, name, args...)
	return err
}

func output(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return string(out), nil
}

func (m *Manager) Remove(id uuid.UUID) error {
	if err := os.RemoveAll(filepath.Join(m.root, id.String())); err != nil {
		return fmt.Errorf("remove plugin installation: %w", err)
	}
	return nil
}

// Execute buffers output until the plugin completes successfully.
func (m *Manager) Execute(ctx context.Context, definition Definition, input io.Reader) ([]byte, error) {
	installed, dir, err := m.current(ctx, definition.ID)
	if err != nil {
		return nil, err
	}
	if installed == nil {
		return nil, apperror.New(apperror.CodeConflict, "Plugin is not installed.")
	}
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	cmd, err := command(ctx, installed.Runtime, dir, definition.Arguments)
	if err != nil {
		return nil, err
	}

	// output uses memory up to the configured limit per conversion; use disk buffering if concurrency grows.
	var buffer bytes.Buffer
	out := &limitedWriter{w: &buffer, remaining: m.max}
	cmd.Stdin = input
	cmd.Stdout = out

	if err := cmd.Run(); err != nil {
		switch {
		case out.exceeded:
			return nil, apperror.New(apperror.CodeRequestTooLarge, "Converted output exceeds the configured limit.")
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			return nil, apperror.Wrap(apperror.CodeUpstreamTimeout, "Plugin execution timed out.", err)
		}
		if _, ok := errors.AsType[*apperror.AppError](err); ok {
			return nil, err
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "Plugin execution failed.", err)
	}

	if buffer.Len() == 0 {
		return nil, apperror.New(apperror.CodeValidationFailed, "Plugin produced empty output.")
	}
	return buffer.Bytes(), nil
}

// command builds the process for a runtime, running inside dir.
func command(ctx context.Context, runtime domain.Runtime, dir string, args []string) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	switch runtime {
	case domain.RuntimePython:
		cmd = exec.CommandContext(ctx, filepath.Join(dir, "venv", "bin", "python"), append([]string{"plugin.py"}, args...)...)
	case domain.RuntimeJava:
		cmd = exec.CommandContext(ctx, "java", append([]string{"-jar", "plugin.jar"}, args...)...)
	case domain.RuntimeBinary:
		cmd = exec.CommandContext(ctx, filepath.Join(dir, "plugin"), args...)
	default:
		return nil, apperror.Default(apperror.CodeInternal)
	}
	cmd.Dir = dir
	return cmd, nil
}

// limitedWriter fails once more than remaining bytes would be written.
type limitedWriter struct {
	w         io.Writer
	remaining int64
	exceeded  bool
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > l.remaining {
		l.exceeded = true
		return 0, apperror.New(apperror.CodeRequestTooLarge, "Converted output exceeds the configured limit.")
	}
	n, err := l.w.Write(p)
	l.remaining -= int64(n)
	return n, err
}
