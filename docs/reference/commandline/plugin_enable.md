<!--[metadata]>
+++
title = "plugin enable"
description = "the plugin enable command description and usage"
keywords = ["plugin, enable"]
[menu.main]
parent = "smn_cli"
+++
<![end-metadata]-->

# plugin enable

    Usage: docker plugin enable PLUGIN

    Enable a plugin

      --help             Print usage

Enables a plugin. The plugin must be installed before it can be enabled,
see [`docker plugin install`](plugin_install.md).


The following example shows that the `no-remove` plugin is currently installed,
but disabled ("inactive"):

```bash
$ docker plugin ls
NAME                	VERSION             ACTIVE
aragunathan/no-remove	latest              false
```
To enable the plugin, use the following command:

```bash
$ docker plugin enable aragunathan/no-remove:latest
```

After the plugin is enabled, it appears as "active" in the list of plugins:

```bash
$ docker plugin ls
NAME                	VERSION             ACTIVE
aragunathan/no-remove	latest              true
```

## Related information

* [plugin ls](plugin_ls.md)
* [plugin disable](plugin_disable.md)
* [plugin inspect](plugin_inspect.md)
* [plugin install](plugin_install.md)
* [plugin rm](plugin_rm.md)
* [plugin set](plugin_set.md)
