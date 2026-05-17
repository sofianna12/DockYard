package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
)

func ImageExistsLocally(imageName string) bool {
	cli := GetClient()
	_, _, err := cli.ImageInspectWithRaw(context.Background(), imageName)
	return err == nil
}

func PullImage(imageName, registryUser, registryPassword string) error {
	if ImageExistsLocally(imageName) {
		return nil
	}

	cli := GetClient()
	ctx := context.Background()

	opts := image.PullOptions{}
	if registryUser != "" && registryPassword != "" {
		authConfig := registry.AuthConfig{
			Username: registryUser,
			Password: registryPassword,
		}
		encoded, err := json.Marshal(authConfig)
		if err != nil {
			return err
		}
		opts.RegistryAuth = base64.URLEncoding.EncodeToString(encoded)
	}

	reader, err := cli.ImagePull(ctx, imageName, opts)
	if err != nil {
		return err
	}
	defer reader.Close()
	io.Copy(io.Discard, reader)
	return nil
}
