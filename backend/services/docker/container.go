package docker

import (
	"context"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

func RunContainer(image string, envVars []string, binds []string, containerPort int, terminalMode bool) (containerID string, port int, err error) {
	cli := GetClient()
	ctx := context.Background()

	hostConfig := &container.HostConfig{Binds: binds}

	containerCfg := &container.Config{Image: image, Env: envVars}

	if terminalMode {
		// Keep the container alive so docker exec can attach a shell.
		containerCfg.Entrypoint = []string{"tail"}
		containerCfg.Cmd = []string{"-f", "/dev/null"}
	} else if containerPort > 0 {
		// Fixed port mapping: use the same port on the host so BASE_URL stays predictable.
		p := nat.Port(fmt.Sprintf("%d/tcp", containerPort))
		hostConfig.PortBindings = nat.PortMap{
			p: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", containerPort)}},
		}
	} else {
		hostConfig.PublishAllPorts = true
	}

	resp, err := cli.ContainerCreate(ctx,
		containerCfg,
		hostConfig,
		&network.NetworkingConfig{},
		nil,
		"",
	)
	if err != nil {
		return "", 0, err
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", 0, err
	}

	if terminalMode {
		return resp.ID, 0, nil
	}

	port, err = GetContainerPort(resp.ID, containerPort)
	return resp.ID, port, err
}

// GetContainerPort returns the host port mapped to containerPort inside the container.
// If containerPort is 0, it returns the first port found (any protocol).
func GetContainerPort(containerID string, containerPort int) (int, error) {
	cli := GetClient()
	info, err := cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return 0, err
	}

	if containerPort > 0 {
		key := fmt.Sprintf("%d/tcp", containerPort)
		for portKey, bindings := range info.NetworkSettings.Ports {
			if string(portKey) == key {
				for _, b := range bindings {
					if p, err := strconv.Atoi(b.HostPort); err == nil {
						return p, nil
					}
				}
			}
		}
		return 0, fmt.Errorf("container port %d not exposed by %s", containerPort, containerID)
	}

	for _, bindings := range info.NetworkSettings.Ports {
		for _, b := range bindings {
			if p, err := strconv.Atoi(b.HostPort); err == nil {
				return p, nil
			}
		}
	}
	return 0, fmt.Errorf("no port found for container %s", containerID)
}

func StopAndRemoveContainer(containerID string) error {
	cli := GetClient()
	ctx := context.Background()
	cli.ContainerStop(ctx, containerID, container.StopOptions{})
	return cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}
