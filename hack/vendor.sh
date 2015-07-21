#!/usr/bin/env bash
set -e

cd "$(dirname "$BASH_SOURCE")/.."
rm -rf vendor/
source 'hack/.vendor-helpers.sh'

# the following lines are in sorted order, FYI
clone git github.com/Sirupsen/logrus v0.8.2 # logrus is a common dependency among multiple deps
clone git github.com/docker/libtrust 9cbd2a1374f46905c68a4eb3694a130610adc62a
clone git github.com/go-check/check 64131543e7896d5bcc6bd5a76287eb75ea96c673
clone git github.com/gorilla/context 14f550f51a
clone git github.com/gorilla/mux e444e69cbd
clone git github.com/kr/pty 5cf931ef8f
clone git github.com/microsoft/hcsshim f674a70f1306dbe20b3a516bedd3285d85db60d9
clone git github.com/mattn/go-sqlite3 b4142c444a8941d0d92b0b7103a24df9cd815e42
clone git github.com/mistifyio/go-zfs v2.1.1
clone git github.com/natefinch/npipe 0938d701e50e580f5925c773055eb6d6b32a0cbc
clone git github.com/tchap/go-patricia v2.1.0
clone git golang.org/x/net 3cffabab72adf04f8e3b01c5baf775361837b5fe https://github.com/golang/net.git
clone hg code.google.com/p/gosqlite 74691fb6f837

#get libnetwork packages
clone git github.com/docker/libnetwork 3daf67270570c1e07e3e3184d46a10f0c5d66f87
clone git github.com/vishvananda/netns 493029407eeb434d0c2d44e02ea072ff2488d322
clone git github.com/vishvananda/netlink 20397a138846e4d6590e01783ed023ed7e1c38a6

# get graph and distribution packages
clone git github.com/docker/distribution cd8ff553b6b1911be23dfeabb73e33108bcbf147
clone git github.com/vbatts/tar-split v0.9.4

clone git github.com/opencontainers/runc v0.0.2 # libcontainer
# libcontainer deps (see src/github.com/docker/libcontainer/update-vendor.sh)
clone git github.com/coreos/go-systemd v2
clone git github.com/godbus/dbus v2
clone git github.com/syndtr/gocapability 66ef2aa7a23ba682594e2b6f74cf40c0692b49fb
clone git github.com/golang/protobuf 655cdfa588ea
clone git github.com/Graylog2/go-gelf 6c62a85f1d47a67f2a5144c0e745b325889a8120

clone git github.com/fluent/fluent-logger-golang v1.0.0
# fluent-logger-golang deps
clone git github.com/philhofer/fwd 899e4efba8eaa1fea74175308f3fae18ff3319fa
clone git github.com/tinylib/msgp 75ee40d2601edf122ef667e2a07d600d4c44490c

# fsnotify
clone git gopkg.in/fsnotify.v1 v1.2.0

clean
