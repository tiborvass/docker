<!--[metadata]>
+++
title = "plugin install"
description = "the plugin install command description and usage"
keywords = ["plugin, install"]
[menu.main]
parent = "smn_cli"
+++
<![end-metadata]-->

# plugin install

    Usage: docker plugin install PLUGIN

    Install a plugin

      --help             Print usage

Installs and enables a plugin. Docker looks first for the plugin on your Docker
host. If the plugin does not exist locally, then the plugin is pulled from
Docker Hub.


The following example installs and enables the `no-remove` plugin:

```bash
$ docker plugin install no-remove
```

After the plugin is installed, it appears in the list of plugins:

```bash
$ docker plugin ls
NAME                VERSION             ACTIVE
no-remove:latest    latest              true
```

## Related information

* [plugin ls](plugin_ls.md)
* [plugin enable](plugin_enable.md)
* [plugin disable](plugin_disable.md)
* [plugin inspect](plugin_inspect.md)
* [plugin rm](plugin_rm.md)
* [plugin set](plugin_set.md)
