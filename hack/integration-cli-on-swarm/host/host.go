package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/filters"
	"github.com/tiborvass/docker/client"
	"github.com/tiborvass/docker/pkg/stdcopy"
)

const (
	defaultStackName       = "integration-cli-on-swarm"
	defaultVolumeName      = "integration-cli-on-swarm"
	defaultMasterImageName = "integration-cli-master"
	defaultWorkerImageName = "integration-cli-worker"
)

func main() {
	if err := xmain(); err != nil {
		logrus.Fatalf("fatal error: %v", err)
	}
}

// xmain can call os.Exit()
func xmain() error {
	// Should we use cobra maybe?
	replicas := flag.Int("replicas", 1, "Number of worker service replica")
	chunks := flag.Int("chunks", 0, "Number of test chunks executed in batch (0 == replicas)")
	pushWorkerImage := flag.String("push-worker-image", "", "Push the worker image to the registry. Required for distribuetd execution. (empty == not to push)")
	shuffle := flag.Bool("shuffle", false, "Shuffle the input so as to mitigate makespan nonuniformity")
	// flags below are rarely used
	randSeed := flag.Int64("rand-seed", int64(0), "Random seed used for shuffling (0 == curent time)")
	filtersFile := flag.String("filters-file", "", "Path to optional file composed of `-check.f` filter strings")
	dryRun := flag.Bool("dry-run", false, "Dry run")
	flag.Parse()
	if *chunks == 0 {
		*chunks = *replicas
	}
	if *randSeed == int64(0) {
		*randSeed = time.Now().UnixNano()
	}
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	if hasStack(cli, defaultStackName) {
		logrus.Infof("Removing stack %s", defaultStackName)
		removeStack(cli, defaultStackName)
	}
	if hasVolume(cli, defaultVolumeName) {
		logrus.Infof("Removing volume %s", defaultVolumeName)
		removeVolume(cli, defaultVolumeName)
	}
	if err = ensureImages(cli, []string{defaultWorkerImageName, defaultMasterImageName}); err != nil {
		return err
	}
	workerImageForStack := defaultWorkerImageName
	if *pushWorkerImage != "" {
		logrus.Infof("Pushing %s to %s", defaultWorkerImageName, *pushWorkerImage)
		if err = pushImage(cli, *pushWorkerImage, defaultWorkerImageName); err != nil {
			return err
		}
		workerImageForStack = *pushWorkerImage
	}
	compose, err := createCompose("", cli, composeOptions{
		Replicas:    *replicas,
		Chunks:      *chunks,
		MasterImage: defaultMasterImageName,
		WorkerImage: workerImageForStack,
		Volume:      defaultVolumeName,
		Shuffle:     *shuffle,
		RandSeed:    *randSeed,
		DryRun:      *dryRun,
	})
	if err != nil {
		return err
	}
	filters, err := filtersBytes(*filtersFile)
	if err != nil {
		return err
	}
	logrus.Infof("Creating volume %s with input data", defaultVolumeName)
	if err = createVolumeWithData(cli,
		defaultVolumeName,
		map[string][]byte{"/input": filters},
		defaultMasterImageName); err != nil {
		return err
	}
	logrus.Infof("Deploying stack %s from %s", defaultStackName, compose)
	defer func() {
		logrus.Infof("NOTE: You may want to inspect or clean up following resources:")
		logrus.Infof(" - Stack: %s", defaultStackName)
		logrus.Infof(" - Volume: %s", defaultVolumeName)
		logrus.Infof(" - Compose file: %s", compose)
		logrus.Infof(" - Master image: %s", defaultMasterImageName)
		logrus.Infof(" - Worker image: %s", workerImageForStack)
	}()
	if err = deployStack(cli, defaultStackName, compose); err != nil {
		return err
	}
	logrus.Infof("The log will be displayed here after some duration."+
		"You can watch the live status via `docker service logs %s_worker`",
		defaultStackName)
	masterContainerID, err := waitForMasterUp(cli, defaultStackName)
	if err != nil {
		return err
	}
	rc, err := waitForContainerCompletion(cli, os.Stdout, os.Stderr, masterContainerID)
	if err != nil {
		return err
	}
	logrus.Infof("Exit status: %d", rc)
	os.Exit(int(rc))
	return nil
}

func ensureImages(cli *client.Client, images []string) error {
	for _, image := range images {
		_, _, err := cli.ImageInspectWithRaw(context.Background(), image)
		if err != nil {
			return fmt.Errorf("could not find image %s, please run `make build-integration-cli-on-swarm`: %v",
				image, err)
		}
	}
	return nil
}

func filtersBytes(optionalFiltersFile string) ([]byte, error) {
	var b []byte
	if optionalFiltersFile == "" {
		tests, err := enumerateTests(".")
		if err != nil {
			return b, err
		}
		b = []byte(strings.Join(tests, "\n") + "\n")
	} else {
		var err error
		b, err = ioutil.ReadFile(optionalFiltersFile)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

func waitForMasterUp(cli *client.Client, stackName string) (string, error) {
	// FIXME(AkihiroSuda): it should retry until master is up, rather than pre-sleeping
	time.Sleep(10 * time.Second)

	fil := filters.NewArgs()
	fil.Add("label", "com.docker.stack.namespace="+stackName)
	// FIXME(AkihiroSuda): we should not rely on internal service naming convention
	fil.Add("label", "com.docker.swarm.service.name="+stackName+"_master")
	masters, err := cli.ContainerList(context.Background(), types.ContainerListOptions{
		All:     true,
		Filters: fil,
	})
	if err != nil {
		return "", err
	}
	if len(masters) == 0 {
		return "", fmt.Errorf("master not running in stack %s?", stackName)
	}
	return masters[0].ID, nil
}

func waitForContainerCompletion(cli *client.Client, stdout, stderr io.Writer, containerID string) (int64, error) {
	stream, err := cli.ContainerLogs(context.Background(),
		containerID,
		types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		})
	if err != nil {
		return 1, err
	}
	stdcopy.StdCopy(stdout, stderr, stream)
	stream.Close()
	return cli.ContainerWait(context.Background(), containerID)
}
