#!/usr/bin/env bash
set -e

cd "$(dirname "$BASH_SOURCE")/.."

# Downloads dependencies into vendor/ directory
mkdir -p vendor
cd vendor

clone() {
	vcs=$1
	pkg=$2
	rev=$3

	pkg_url=https://$pkg
	target_dir=src/$pkg

	echo -n "$pkg @ $rev: "

	if [ -d $target_dir ]; then
		echo -n 'rm old, '
		rm -fr $target_dir
	fi

	echo -n 'clone, '
	case $vcs in
		git)
			git clone --quiet --no-checkout $pkg_url $target_dir
			( cd $target_dir && git reset --quiet --hard $rev )
			;;
		hg)
			hg clone --quiet --updaterev $rev $pkg_url $target_dir
			;;
	esac

	echo -n 'rm VCS, '
	( cd $target_dir && rm -rf .{git,hg} )

	echo -n 'rm vendor, '
	( cd $target_dir && rm -rf vendor Godeps/_workspace )

	echo done
}

# the following lines are in sorted order, FYI
clone git github.com/Sirupsen/logrus v0.7.3 # logrus is a common dependency among multiple deps
clone git github.com/docker/libtrust c54fbb67c1f1e68d7d6f8d2ad7c9360404616a41
clone git github.com/garyburd/redigo 2668ae2944701f73206a228638407926dea859b1
clone git github.com/go-check/check 64131543e7896d5bcc6bd5a76287eb75ea96c673
clone git github.com/go-fsnotify/fsnotify v1.2.0
clone git github.com/gorilla/context 14f550f51a
clone git github.com/gorilla/mux e444e69cbd
clone git github.com/kr/pty 5cf931ef8f
clone git github.com/jlhawn/go-crypto cd738dde20f0b3782516181b0866c9bb9db47401
clone git github.com/mistifyio/go-zfs v2.1.0
clone git github.com/tchap/go-patricia v2.1.0
clone hg code.google.com/p/go.net 84a4013f96e0
clone hg code.google.com/p/gosqlite 74691fb6f837

# get distribution packages
clone git github.com/dmcgowan/distribution 006ddd8283013d392d0e19bdf1adda5bac6612fe
rm -rf src/github.com/dmcgowan/distribution/Godeps

rm -rf src/github.com/docker/distribution
mkdir -p src/github.com/docker
mv src/github.com/dmcgowan/distribution src/github.com/docker/

clone git github.com/golang/net 1dfe7915deaf
rm -rf src/golang.org/x/net
mkdir -p src/golang.org/x
mv src/github.com/golang/net src/golang.org/x/

clone git github.com/docker/libcontainer a37b2a4f152e2a1c9de596f54c051cb889de0691
# libcontainer deps (see src/github.com/docker/libcontainer/update-vendor.sh)
clone git github.com/coreos/go-systemd v2
clone git github.com/godbus/dbus v2
clone git github.com/syndtr/gocapability 66ef2aa7a23ba682594e2b6f74cf40c0692b49fb
