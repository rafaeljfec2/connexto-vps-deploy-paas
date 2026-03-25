package docker

import (
	"context"
	"fmt"
	"time"
)

type BuildOptions struct {
	BuildArgs map[string]string
	Target    string
}

func (d *Client) Build(ctx context.Context, workDir, dockerfile, tag string, output chan<- string) error {
	return d.BuildWithOptions(ctx, workDir, dockerfile, tag, nil, output)
}

func (d *Client) BuildWithOptions(ctx context.Context, workDir, dockerfile, tag string, opts *BuildOptions, output chan<- string) error {
	d.logger.Info("Building Docker image", "workDir", workDir, "dockerfile", dockerfile, "tag", tag)

	d.executor.SetWorkDir(workDir)

	var args []string
	if d.buildxAvailable {
		args = []string{
			"buildx", "build",
			"--builder", "default",
			"--load",
			"-t", tag,
			"-f", dockerfile,
		}
	} else {
		args = []string{
			"build",
			"-t", tag,
			"-f", dockerfile,
		}
	}

	if opts != nil {
		for k, v := range opts.BuildArgs {
			args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
		}
		if opts.Target != "" {
			args = append(args, "--target", opts.Target)
		}
	}

	args = append(args, ".")

	if output != nil {
		err := d.executor.RunWithStreamingTimeout(ctx, 15*time.Minute, output, "docker", args...)
		if err != nil {
			d.logger.Error("Docker build failed", "tag", tag, "workDir", workDir, "error", err)
			return fmt.Errorf("docker build failed: %w", err)
		}
		return nil
	}

	_, err := d.executor.RunWithTimeout(ctx, 15*time.Minute, "docker", args...)
	if err != nil {
		d.logger.Error("Docker build failed", "tag", tag, "workDir", workDir, "error", err)
		return fmt.Errorf("docker build failed: %w", err)
	}

	return nil
}
