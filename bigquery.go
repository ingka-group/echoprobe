// Copyright © 2024 Ingka Holding B.V. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// You may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package echoprobe

import (
	"context"
	"fmt"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultBigqueryImage = "ghcr.io/goccy/bigquery-emulator:latest"
	bqMountPath          = "/mnt/data.yaml"
	bqHttpPort           = "9050/tcp"
	bqGrpcPort           = "9060/tcp"
	bqProject            = "test"
)

type BigqueryEmulatorContainer struct {
	testcontainers.Container

	BqHost     string
	BqRestPort int
	BqGrpcPort int
}

func setupBigqueryEmulator(ctx context.Context, image string, dataPath string) (*BigqueryEmulatorContainer, error) {
	if image == "" {
		image = defaultBigqueryImage
	}

	executionPath, err := testpath()
	if err != nil {
		return nil, err
	}

	req := testcontainers.ContainerRequest{
		Image: image,
		HostConfigModifier: func(config *container.HostConfig) {
			config.Mounts = append(config.Mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: fmt.Sprintf("%s/%s", executionPath, dataPath),
				Target: bqMountPath,
			})
		},
		Cmd: []string{
			fmt.Sprintf("--project=%s", bqProject),
			fmt.Sprintf("--data-from-yaml=%s", bqMountPath),
		},
		ExposedPorts: []string{bqHttpPort, bqGrpcPort},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(bqGrpcPort),
			wait.ForListeningPort(bqHttpPort),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	hostIP, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedHttpPort, err := container.MappedPort(ctx, bqHttpPort)
	if err != nil {
		return nil, err
	}

	mappedGrpcPort, err := container.MappedPort(ctx, bqGrpcPort)
	if err != nil {
		return nil, err
	}

	return &BigqueryEmulatorContainer{
		Container:  container,
		BqHost:     hostIP,
		BqRestPort: int(mappedHttpPort.Num()),
		BqGrpcPort: int(mappedGrpcPort.Num()),
	}, nil
}
