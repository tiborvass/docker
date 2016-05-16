#!/bin/sh

remote_name=tiborvass/no-remove

set -e
name="$1"
echo $name
mkdir -p /var/lib/docker/plugins/$name/rootfs

id=$(docker create "$remote_name" true)
docker export "$id" | tar -x -C /var/lib/docker/plugins/$name/rootfs
docker rm -vf "$id"


#Create a sample manifest file
cat <<EOF > /var/lib/docker/plugins/$name/manifest.json
{
	"manifestVersion": "0.1",
	"description": "A test plugin for Docker",
	"documentation": "https://docs.docker.com/engine/extend/plugins/",
	"entrypoint": ["plugin-no-remove", "/data"],
	"interface" : {
		"types": ["docker.volumedriver/1.0"],
		"socket": "plugins.sock"
	},

	"network": {
		"type": "host"
	},

	"arguments": [
		{
			"name": "arg1",
			"description": "a command line argument",
			"value": "value1"
		}
	],


	"mounts": [
		{
			"source": "/data",
			"destination": "/data",
			"type": "bind",
			"options": ["shared", "rbind"],
			"name": "mountpoint-prefix",
			"description": "host path to folder holding volumes",
			"settable": true
		},
		{
			"destination": "/run/docker/plugins",
			"type": "docker.plugin.runtime"
		},
		{
			"destination": "/state",
			"type": "docker.plugin.state"
		},
		{
			"destination": "/foobar",
			"type": "tmpfs"
		}
	],

	"env": [
		{
			"name": "DEBUG",
			"value": "1",
			"description": "If set, prints debug messages",
			"settable": true
		}
	],

	"devices": [
		{
			"name": "device",
			"description": "a host device to mount",
			"path": "/dev/cpu_dma_latency",
			"settable": true
		}
	],

	"capabilities": ["CAP_SYS_ADMIN"]
}
EOF
