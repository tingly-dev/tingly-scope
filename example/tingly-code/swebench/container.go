package swebench

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
)

// ContainerManager manages Docker containers for SWEbench tasks
type ContainerManager struct {
	cli    *docker.Client
	config *Config
}

// NewContainerManager creates a new container manager
func NewContainerManager(cfg *Config) (*ContainerManager, error) {
	cli, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &ContainerManager{
		cli:    cli,
		config: cfg,
	}, nil
}

// Close closes the Docker client connection
func (cm *ContainerManager) Close() error {
	return nil
}

// ContainerRunOptions controls how a container is run
type ContainerRunOptions struct {
	// Task is the SWEbench task
	Task *Task

	// WorkDir is the host working directory
	WorkDir string

	// AgentBinary is the path to tingly-code binary for the container platform
	AgentBinary string

	// ConfigPath is the path to the config file
	ConfigPath string

	// Image is the Docker image to use (overrides default)
	Image string

	// KeepContainer keeps the container after execution
	KeepContainer bool

	// Progress reports progress
	Progress func(msg string)

	// OutputWriter receives container output
	OutputWriter io.Writer
}

// RunTaskInContainer runs a SWEbench task inside a Docker container
func (cm *ContainerManager) RunTaskInContainer(ctx context.Context, opts ContainerRunOptions) (*RunResult, error) {
	result := &RunResult{
		TaskID: opts.Task.TaskID,
		Status: StatusRunning,
	}

	startTime := time.Now()

	// Use pre-built SWEbench image (linux/amd64)
	image := opts.Image
	if image == "" {
		image = getSWEbenchImageName(opts.Task.TaskID)
	}

	if opts.Progress != nil {
		opts.Progress(fmt.Sprintf("Using SWEbench image: %s", image))
	}

	// Pull image if needed
	if err := cm.ensureImage(ctx, image, opts.Progress); err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("failed to pull image: %w", err)
		return result, fmt.Errorf(result.Error)
	}

	// Create container
	containerName := fmt.Sprintf("swebench-%s", strings.ReplaceAll(opts.Task.TaskID, "/", "-"))

	// Prepare working directory
	workDir := opts.WorkDir
	if workDir == "" {
		workDir = cm.config.WorkDir
	}
	os.MkdirAll(workDir, 0755)

	containerConfig := docker.CreateContainerOptions{
		Name: containerName,
		Config: &docker.Config{
			Image: image,
			Cmd:   []string{"sleep", "3600"}, // Keep alive
			Tty:   false,
			Env:   []string{"DEBIAN_FRONTEND=noninteractive"},
		},
		HostConfig: &docker.HostConfig{
			Binds: []string{
				fmt.Sprintf("%s:/output", workDir),
			},
		},
		Platform: "linux/amd64",
	}

	if opts.Progress != nil {
		opts.Progress(fmt.Sprintf("Creating container %s...", containerName))
	}

	container, err := cm.cli.CreateContainer(containerConfig)
	if err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("failed to create container: %w", err)
		return result, fmt.Errorf(result.Error)
	}

	// Cleanup function
	defer func() {
		if !opts.KeepContainer && !cm.config.KeepContainer {
			cm.cli.RemoveContainer(docker.RemoveContainerOptions{
				ID:    container.ID,
				Force: true,
			})
		}
	}()

	// Start container
	if opts.Progress != nil {
		opts.Progress("Starting container...")
	}

	if err := cm.cli.StartContainer(container.ID, nil); err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("failed to start container: %w", err)
		return result, fmt.Errorf(result.Error)
	}

	// Copy tingly-code binary into container
	if opts.AgentBinary != "" {
		if opts.Progress != nil {
			opts.Progress("Copying tingly-code binary into container...")
		}
		if err := cm.copyFileToContainer(ctx, container.ID, opts.AgentBinary, "/usr/local/bin/tingly-code"); err != nil {
			result.Status = StatusFailed
			result.Error = fmt.Sprintf("failed to copy binary: %w", err)
			return result, fmt.Errorf(result.Error)
		}
		// Make executable
		cm.execInContainer(ctx, container.ID, []string{"chmod", "+x", "/usr/local/bin/tingly-code"}, nil, nil)
	}

	// Copy config file into container
	if opts.ConfigPath != "" {
		if opts.Progress != nil {
			opts.Progress("Copying config file into container...")
		}
		if err := cm.copyFileToContainer(ctx, container.ID, opts.ConfigPath, "/root/config.toml"); err != nil {
			result.Status = StatusFailed
			result.Error = fmt.Sprintf("failed to copy config: %w", err)
			return result, fmt.Errorf(result.Error)
		}
	}

	// Run tingly-code agent in container
	if opts.Progress != nil {
		opts.Progress("Running tingly-code agent...")
	}

	prompt := cm.buildPrompt(opts.Task)

	var agentOutput strings.Builder
	outputWriter := io.MultiWriter(&agentOutput, opts.OutputWriter)

	// Build agent command: use -c only if config was explicitly provided
	var agentCmd string
	if opts.ConfigPath != "" {
		// Config was explicitly provided, use it
		agentCmd = fmt.Sprintf("cd /testbed && /usr/local/bin/tingly-code auto -c /root/config.toml %s", escapeShellArg(prompt))
	} else {
		// No config specified, let tingly-code use its default config discovery
		agentCmd = fmt.Sprintf("cd /testbed && /usr/local/bin/tingly-code auto %s", escapeShellArg(prompt))
	}

	if err := cm.execInContainer(ctx, container.ID, []string{"bash", "-cl", agentCmd}, outputWriter, opts.Progress); err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("agent execution failed: %w", err)
		result.Output = agentOutput.String()
		return result, fmt.Errorf(result.Error)
	}

	result.Output = agentOutput.String()

	// Run tests to verify the fix
	if opts.Progress != nil {
		opts.Progress("Running tests to verify fix...")
	}

	testOutput, err := cm.runTestsInContainer(ctx, container.ID, opts.Progress)
	result.TestOutput = testOutput

	// Check if tests passed
	result.Passed = err == nil || cm.checkTestsPassed(testOutput)

	result.Status = StatusCompleted
	result.Duration = time.Since(startTime)

	return result, nil
}

// copyFileToContainer copies a file from host to container
func (cm *ContainerManager) copyFileToContainer(ctx context.Context, containerID, srcPath, destPath string) error {
	// Read source file
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Create a tar archive in memory
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	hdr := &tar.Header{
		Name: filepath.Base(destPath),
		Mode: 0755,
		Size: int64(len(data)),
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("failed to write file data: %w", err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Create upload options
	uploadOpts := docker.UploadToContainerOptions{
		Path:        filepath.Dir(destPath),
		InputStream: &buf,
	}

	// Upload to container
	return cm.cli.UploadToContainer(containerID, uploadOpts)
}

// runTestsInContainer runs tests inside the container
func (cm *ContainerManager) runTestsInContainer(ctx context.Context, containerID string, progress func(msg string)) (string, error) {
	var output strings.Builder

	// The testbed is in /testbed
	testCmd := "cd /testbed && python -m pytest -xvs 2>&1 || true"

	if progress != nil {
		progress(fmt.Sprintf("Running tests..."))
	}

	if err := cm.execInContainer(ctx, containerID, []string{
		"sh", "-c", testCmd,
	}, &output, nil); err != nil {
		return output.String(), err
	}

	return output.String(), nil
}

// execInContainer executes a command inside the container
func (cm *ContainerManager) execInContainer(ctx context.Context, containerID string, cmd []string, output io.Writer, progress func(msg string)) error {
	if progress != nil {
		progress(fmt.Sprintf("Executing: %s", strings.Join(cmd, " ")))
	}

	// Create exec
	execOpts := docker.CreateExecOptions{
		Container:    containerID,
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	exec, err := cm.cli.CreateExec(execOpts)
	if err != nil {
		return fmt.Errorf("failed to create exec: %w", err)
	}

	// Start exec and get output
	startOpts := docker.StartExecOptions{
		OutputStream: output,
		ErrorStream:  output,
	}

	if err := cm.cli.StartExec(exec.ID, startOpts); err != nil {
		return fmt.Errorf("failed to start exec: %w", err)
	}

	// Wait for exec to complete and check exit code
	execInfo, err := cm.cli.InspectExec(exec.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect exec: %w", err)
	}

	if execInfo.ExitCode != 0 {
		return fmt.Errorf("exec failed with exit code %d", execInfo.ExitCode)
	}

	return nil
}

// ensureImage pulls a Docker image if it doesn't exist locally
func (cm *ContainerManager) ensureImage(ctx context.Context, image string, progress func(msg string)) error {
	_, err := cm.cli.InspectImage(image)
	if err == nil {
		return nil // Image exists
	}

	if progress != nil {
		progress(fmt.Sprintf("Pulling image %s...", image))
	}

	pullOpts := docker.PullImageOptions{
		Repository:   image,
		OutputStream: os.Stderr,
		Platform:     "linux/amd64",
	}

	if err := cm.cli.PullImage(pullOpts, docker.AuthConfiguration{}); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	return nil
}

// getSWEbenchImageName converts a task ID to SWEbench image name
func getSWEbenchImageName(taskID string) string {
	// Replace __ with _1776_ (Docker doesn't allow double underscore)
	idDockerCompatible := strings.ReplaceAll(taskID, "__", "_1776_")
	return fmt.Sprintf("swebench/sweb.eval.x86_64.%s:latest", strings.ToLower(idDockerCompatible))
}

// buildPrompt creates the agent prompt from the task
func (cm *ContainerManager) buildPrompt(task *Task) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("You are working on the repository at /testbed\n"))
	prompt.WriteString(fmt.Sprintf("Base commit: %s\n\n", task.BaseCommit))
	prompt.WriteString("Please fix the following issue:\n\n")
	prompt.WriteString(task.ProblemStatement)
	prompt.WriteString("\n\n")

	if len(task.Hints) > 0 {
		prompt.WriteString("Hints:\n")
		for _, hint := range task.Hints {
			prompt.WriteString(fmt.Sprintf("- %s\n", hint))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("Steps:\n")
	prompt.WriteString("1. Analyze the problem\n")
	prompt.WriteString("2. Find the relevant code in /testbed\n")
	prompt.WriteString("3. Implement the fix\n")
	prompt.WriteString("4. Run tests to verify\n")
	prompt.WriteString("5. Call job_done when complete")

	return prompt.String()
}

// checkTestsPassed parses test output to determine if tests passed
func (cm *ContainerManager) checkTestsPassed(output string) bool {
	passedPatterns := []string{"passed", "PASS", "OK", "success"}

	lowerOutput := strings.ToLower(output)
	for _, pattern := range passedPatterns {
		if strings.Contains(lowerOutput, strings.ToLower(pattern)) {
			if !strings.Contains(lowerOutput, "failed") &&
				!strings.Contains(lowerOutput, "error") &&
				!strings.Contains(lowerOutput, "FAIL") {
				return true
			}
		}
	}

	return false
}

// GetContainerLogs gets logs from a container
func (cm *ContainerManager) GetContainerLogs(ctx context.Context, containerID string) (string, error) {
	opts := docker.LogsOptions{
		Container:    containerID,
		OutputStream: os.Stdout,
		ErrorStream:  os.Stderr,
		Stdout:       true,
		Stderr:       true,
		Tail:         "100",
	}

	var output strings.Builder
	opts.OutputStream = &output
	opts.ErrorStream = &output

	if err := cm.cli.Logs(opts); err != nil {
		return "", err
	}

	return output.String(), nil
}

// StopContainer stops a running container
func (cm *ContainerManager) StopContainer(ctx context.Context, containerID string) error {
	timeout := 10 * time.Second
	return cm.cli.StopContainer(containerID, uint(timeout.Seconds()))
}

// ListContainers lists all SWEbench containers
func (cm *ContainerManager) ListContainers(ctx context.Context) ([]docker.APIContainers, error) {
	containers, err := cm.cli.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		return nil, err
	}

	var swebenchContainers []docker.APIContainers
	for _, c := range containers {
		for _, name := range c.Names {
			if strings.HasPrefix(strings.TrimPrefix(name, "/"), "swebench-") {
				swebenchContainers = append(swebenchContainers, c)
				break
			}
		}
	}

	return swebenchContainers, nil
}

// escapeShellArg escapes a shell argument
func escapeShellArg(arg string) string {
	// Simple escaping - wrap in quotes and escape existing quotes
	return "'" + strings.ReplaceAll(arg, "'", "'\"'\"'") + "'"
}

// truncateString truncates a string to max length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen > 3 {
		return s[:maxLen-3] + "..."
	}
	return s[:maxLen]
}
