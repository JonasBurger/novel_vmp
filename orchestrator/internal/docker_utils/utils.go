package dockerutils

import (
	"context"
	"io"
	"log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

var ApiClient = newClient()

func newClient() *client.Client {
	apiClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal(err)
	}
	apiClient.NegotiateAPIVersion(context.TODO())
	return apiClient
}

func Close() {
	ApiClient.Close()
}

func RunContainer(imageName string, containerName string, environmentVariables []string) (id string) {
	log.Println("Running container: ", imageName)

	containerConfig := &container.Config{
		Image:    imageName,
		Hostname: containerName,
		Env:      environmentVariables,
	}

	hostConfig := &container.HostConfig{
		NetworkMode: "host",
	}

	res, err := ApiClient.ContainerCreate(context.TODO(), containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		log.Fatal(err)
	}
	for _, warning := range res.Warnings {
		log.Printf("Warning (%v): %v\n", imageName, warning)
	}

	err = ApiClient.ContainerStart(context.TODO(), res.ID, container.StartOptions{})
	if err != nil {
		log.Fatal(err)
	}

	id = res.ID

	go func() {
		// logging
		out, err := ApiClient.ContainerLogs(context.Background(), id, container.LogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
		if err != nil {
			log.Fatal(err)
		}
		io.Copy(log.Writer(), out)
		log.Printf("%v: container exited", imageName)

	}()

	return id
}

func StopContainer(id string) {
	log.Println("Stopping container: ", id)
	err := ApiClient.ContainerStop(context.TODO(), id, container.StopOptions{})
	if err != nil {
		log.Fatal(err)
	}
}

func RemoveContainer(id string) {
	log.Println("Removing container: ", id)
	err := ApiClient.ContainerRemove(context.TODO(), id, container.RemoveOptions{
		RemoveVolumes: true,
	})
	if err != nil {
		log.Fatal(err)
	}
}
