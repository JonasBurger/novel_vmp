package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"my.org/novel_vmp/internal/config"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/ory/dockertest/v3/docker/pkg/archive"
	"github.com/spf13/viper"
)

func main() {
	config.LoadViperConfig()
	ctx := context.TODO()

	apiClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	defer apiClient.Close()

	apiClient.NegotiateAPIVersion(ctx)

	currnetDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	currnetDir += "/"
	println("currentDir: " + currnetDir)

	scanners := viper.GetStringSlice("scanners")
	for _, scannerPath := range scanners {
		scannerName := filepath.Base(scannerPath)

		if len(os.Args) > 1 && os.Args[1] != scannerName {
			continue
		}

		fmt.Println("================================================================")
		fmt.Printf("Building scanner %v at %v\n", scannerName, scannerPath)

		dockerBuildContext, err := archive.TarWithOptions(currnetDir, &archive.TarOptions{})
		if err != nil {
			panic(err)
		}

		buildOptions := types.ImageBuildOptions{
			Tags:       []string{fmt.Sprintf("%v:latest", scannerName)},
			Dockerfile: scannerPath + "Dockerfile",
			Remove:     true,
		}

		buildResponse, err := apiClient.ImageBuild(ctx, dockerBuildContext, buildOptions)
		if err != nil {
			println("Error: ", err)
			panic(err)
		}
		defer buildResponse.Body.Close()
		scanner := bufio.NewScanner(buildResponse.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			var result map[string]string
			err := json.Unmarshal(line, &result)
			if err != nil {
				continue
			}
			stream, ok := result["stream"]
			if !ok {
				print(string(line))
			}
			fmt.Print(stream)
		}
		if err := scanner.Err(); err != nil {
			println("Error: ", err)
			panic(err)
		}

	}
}
