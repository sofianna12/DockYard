package docker

import (
	"log"
	"sync"

	"github.com/docker/docker/client"
)

var (
	instance *client.Client
	once     sync.Once
)

func GetClient() *client.Client {
	once.Do(func() {
		var err error
		instance, err = client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			log.Fatal("Failed to create Docker client:", err)
		}
	})
	return instance
}
