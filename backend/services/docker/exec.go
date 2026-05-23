package docker

import (
	"context"
	"io"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

type ExecSession struct {
	ID     string
	Conn   io.ReadWriteCloser
	Reader io.Reader
}

// ExecTTY creates and starts an interactive TTY exec session inside a running container.
func ExecTTY(containerID string) (*ExecSession, error) {
	cli := GetClient()
	ctx := context.Background()

	execID, err := cli.ContainerExecCreate(ctx, containerID, dockertypes.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{"sh", "-i"},
	})
	if err != nil {
		return nil, err
	}

	resp, err := cli.ContainerExecAttach(ctx, execID.ID, dockertypes.ExecStartCheck{Tty: true})
	if err != nil {
		return nil, err
	}

	return &ExecSession{
		ID:     execID.ID,
		Conn:   resp.Conn,
		Reader: resp.Reader,
	}, nil
}

// ResizeExecTTY resizes the TTY of an exec session.
func ResizeExecTTY(execID string, rows, cols uint) error {
	cli := GetClient()
	return cli.ContainerExecResize(context.Background(), execID, container.ResizeOptions{
		Height: rows,
		Width:  cols,
	})
}
