<!--[metadata]>
+++
title = "plugin inspect"
description = "The plugin inspect command description and usage"
keywords = ["plugin, inspect"]
[menu.main]
parent = "smn_cli"
+++
<![end-metadata]-->

# plugin inspect

    Usage: docker plugin inspect PLUGIN

    Return low-level information about a plugin

      --help              Print usage


Returns information about a plugin. By default, this command renders all results
in a JSON array.

Example output:

```bash
$ docker plugin inspect no-remove:latest
```
```JSON
{
    "manifestVersion": "0.1",
    "description": "A test plugin for Docker",
    "documentation": "https://docs.docker.com/engine/extend/plugins/",
    "entrypoint": ["plugin-no-remove", "/data"],
    "interface" : {
        "types": ["docker.volumedriver/1.0"],
        "socket": "/var/run/docker/plugins/$name.sock"
    },
    "network": {
        "type": "host"
    },
    "arguments": [
        {
            "name": "arg1",
            "description": "a command line argument",
            "value": "value1"
        },
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
            "settable": true,
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
```
(output formatted for readability)



## Related information

* [plugin ls](plugin_ls.md)
* [plugin enable](plugin_enable.md)
* [plugin disable](plugin_disable.md)
* [plugin install](plugin_install.md)
* [plugin rm](plugin_rm.md)
* [plugin set](plugin_set.md)
