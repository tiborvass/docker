package container // import "github.com/tiborvass/docker/integration/container"

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/tiborvass/docker/api/types"
	"github.com/tiborvass/docker/api/types/container"
	"github.com/tiborvass/docker/api/types/mount"
	"github.com/tiborvass/docker/api/types/network"
	"github.com/tiborvass/docker/client"
	"github.com/tiborvass/docker/integration-cli/daemon"
	"github.com/tiborvass/docker/pkg/stdcopy"
	"github.com/tiborvass/docker/pkg/system"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/gotestyourself/gotestyourself/skip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerShmNoLeak(t *testing.T) {
	t.Parallel()
	d := daemon.New(t, "docker", "dockerd", daemon.Config{})
	client, err := d.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	d.StartWithBusybox(t)
	defer d.Stop(t)

	ctx := context.Background()
	cfg := container.Config{
		Image: "busybox",
		Cmd:   []string{"top"},
	}

	ctr, err := client.ContainerCreate(ctx, &cfg, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	defer client.ContainerRemove(ctx, ctr.ID, types.ContainerRemoveOptions{Force: true})

	if err := client.ContainerStart(ctx, ctr.ID, types.ContainerStartOptions{}); err != nil {
		t.Fatal(err)
	}

	// this should recursively bind mount everything in the test daemons root
	// except of course we are hoping that the previous containers /dev/shm mount did not leak into this new container
	hc := container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:        mount.TypeBind,
				Source:      d.Root,
				Target:      "/testdaemonroot",
				BindOptions: &mount.BindOptions{Propagation: mount.PropagationRPrivate}},
		},
	}
	cfg.Cmd = []string{"/bin/sh", "-c", fmt.Sprintf("mount | grep testdaemonroot | grep containers | grep %s", ctr.ID)}
	cfg.AttachStdout = true
	cfg.AttachStderr = true
	ctrLeak, err := client.ContainerCreate(ctx, &cfg, &hc, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	attach, err := client.ContainerAttach(ctx, ctrLeak.ID, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := client.ContainerStart(ctx, ctrLeak.ID, types.ContainerStartOptions{}); err != nil {
		t.Fatal(err)
	}

	buf := bytes.NewBuffer(nil)

	if _, err := stdcopy.StdCopy(buf, buf, attach.Reader); err != nil {
		t.Fatal(err)
	}

	out := bytes.TrimSpace(buf.Bytes())
	if !bytes.Equal(out, []byte{}) {
		t.Fatalf("mount leaked: %s", string(out))
	}
}

func TestContainerNetworkMountsNoChown(t *testing.T) {
	// chown only applies to Linux bind mounted volumes; must be same host to verify
	skip.If(t, testEnv.DaemonInfo.OSType != "linux" || !testEnv.IsLocalDaemon())

	defer setupTest(t)()

	ctx := context.Background()

	tmpDir := fs.NewDir(t, "network-file-mounts", fs.WithMode(0755), fs.WithFile("nwfile", "network file bind mount", fs.WithMode(0644)))
	defer tmpDir.Remove()

	tmpNWFileMount := tmpDir.Join("nwfile")

	config := container.Config{
		Image: "busybox",
	}
	hostConfig := container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   "bind",
				Source: tmpNWFileMount,
				Target: "/etc/resolv.conf",
			},
			{
				Type:   "bind",
				Source: tmpNWFileMount,
				Target: "/etc/hostname",
			},
			{
				Type:   "bind",
				Source: tmpNWFileMount,
				Target: "/etc/hosts",
			},
		},
	}

	cli, err := client.NewEnvClient()
	require.NoError(t, err)
	defer cli.Close()

	ctrCreate, err := cli.ContainerCreate(ctx, &config, &hostConfig, &network.NetworkingConfig{}, "")
	require.NoError(t, err)
	// container will exit immediately because of no tty, but we only need the start sequence to test the condition
	err = cli.ContainerStart(ctx, ctrCreate.ID, types.ContainerStartOptions{})
	require.NoError(t, err)

	// Check that host-located bind mount network file did not change ownership when the container was started
	// Note: If the user specifies a mountpath from the host, we should not be
	// attempting to chown files outside the daemon's metadata directory
	// (represented by `daemon.repository` at init time).
	// This forces users who want to use user namespaces to handle the
	// ownership needs of any external files mounted as network files
	// (/etc/resolv.conf, /etc/hosts, /etc/hostname) separately from the
	// daemon. In all other volume/bind mount situations we have taken this
	// same line--we don't chown host file content.
	// See GitHub PR 34224 for details.
	statT, err := system.Stat(tmpNWFileMount)
	require.NoError(t, err)
	assert.Equal(t, uint32(0), statT.UID(), "bind mounted network file should not change ownership from root")
}
