package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/tiborvass/docker/client"
)

func system(commands [][]string) error {
	for _, c := range commands {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func pushImage(_ *client.Client, remote, local string) error {
	// FIXME: eliminate os/exec (but it is hard to pass auth without os/exec ...)
	return system([][]string{
		{"docker", "image", "tag", local, remote},
		{"docker", "image", "push", remote},
	})
}

func deployStack(_ *client.Client, stackName, composeFilePath string) error {
	// FIXME: eliminate os/exec (but stack is implemented in CLI ...)
	return system([][]string{
		{"docker", "stack", "deploy",
			"--compose-file", composeFilePath,
			"--with-registry-auth",
			stackName},
	})
}

func hasStack(_ *client.Client, stackName string) bool {
	// FIXME: eliminate os/exec (but stack is implemented in CLI ...)
	out, err := exec.Command("docker", "stack", "ls").CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("`docker stack ls` failed with: %s", string(out)))
	}
	// FIXME: not accurate
	return strings.Contains(string(out), stackName)
}

func removeStack(_ *client.Client, stackName string) error {
	// FIXME: eliminate os/exec (but stack is implemented in CLI ...)
	if err := system([][]string{
		{"docker", "stack", "rm", stackName},
	}); err != nil {
		return err
	}
	// FIXME
	time.Sleep(10 * time.Second)
	return nil
}
