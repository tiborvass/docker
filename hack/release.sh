#!/usr/bin/env bash
set -e

# This script looks for bundles built by make.sh, and releases them on a
# public S3 bucket.
#
# Bundles should be available for the VERSION string passed as argument.
#
# The correct way to call this script is inside a container built by the
# official Dockerfile at the root of the Docker source code. The Dockerfile,
# make.sh and release.sh should all be from the same source code revision.

set -o pipefail

# Print a usage message and exit.
usage() {
	cat >&2 <<'EOF'
To run, I need:
- to be in a container generated by the Dockerfile at the top of the Docker
  repository;
- to be provided with the location of an S3 bucket and path, in
  environment variables AWS_S3_BUCKET and AWS_S3_BUCKET_PATH (default: '');
- to be provided with AWS credentials for this S3 bucket, in environment
  variables AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY;
- a generous amount of good will and nice manners.
The canonical way to run me is to run the image produced by the Dockerfile: e.g.:"

docker run -e AWS_S3_BUCKET=test.docker.com \
           -e AWS_ACCESS_KEY_ID     \
           -e AWS_SECRET_ACCESS_KEY \
           -e AWS_DEFAULT_REGION    \
           -it --privileged         \
           docker ./hack/release.sh
EOF
	exit 1
}

[ "$AWS_S3_BUCKET" ] || usage
[ "$AWS_ACCESS_KEY_ID" ] || usage
[ "$AWS_SECRET_ACCESS_KEY" ] || usage
[ -d /go/src/github.com/moby/moby-core ] || usage
cd /go/src/github.com/moby/moby-core
[ -x hack/make.sh ] || usage

export AWS_DEFAULT_REGION
: ${AWS_DEFAULT_REGION:=us-west-1}

AWS_CLI=${AWS_CLI:-'aws'}

RELEASE_BUNDLES=(
	binary
	cross
	tgz
)

if [ "$1" != '--release-regardless-of-test-failure' ]; then
	RELEASE_BUNDLES=(
		test-unit
		"${RELEASE_BUNDLES[@]}"
		test-integration-cli
	)
fi

VERSION=$(< VERSION)
BUCKET=$AWS_S3_BUCKET
BUCKET_PATH=$BUCKET
[[ -n "$AWS_S3_BUCKET_PATH" ]] && BUCKET_PATH+=/$AWS_S3_BUCKET_PATH

if command -v git &> /dev/null && git rev-parse &> /dev/null; then
	if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
		echo "You cannot run the release script on a repo with uncommitted changes"
		usage
	fi
fi

# These are the 2 keys we've used to sign the deb's
#   release (get.docker.com)
#	GPG_KEY="36A1D7869245C8950F966E92D8576A8BA88D21E9"
#   test    (test.docker.com)
#	GPG_KEY="740B314AE3941731B942C66ADF4FD13717AAD7D6"

setup_s3() {
	echo "Setting up S3"
	# Try creating the bucket. Ignore errors (it might already exist).
	$AWS_CLI s3 mb "s3://$BUCKET" 2>/dev/null || true
	# Check access to the bucket.
	$AWS_CLI s3 ls "s3://$BUCKET" >/dev/null
	# Make the bucket accessible through website endpoints.
	$AWS_CLI s3 website --index-document index --error-document error "s3://$BUCKET"
}

# write_to_s3 uploads the contents of standard input to the specified S3 url.
write_to_s3() {
	DEST=$1
	F=`mktemp`
	cat > "$F"
	$AWS_CLI s3 cp --acl public-read --content-type 'text/plain' "$F" "$DEST"
	rm -f "$F"
}

s3_url() {
	case "$BUCKET" in
		get.docker.com|test.docker.com|experimental.docker.com)
			echo "https://$BUCKET_PATH"
			;;
		*)
			BASE_URL="http://${BUCKET}.s3-website-${AWS_DEFAULT_REGION}.amazonaws.com"
			if [[ -n "$AWS_S3_BUCKET_PATH" ]] ; then
				echo "$BASE_URL/$AWS_S3_BUCKET_PATH"
			else
				echo "$BASE_URL"
			fi
			;;
	esac
}

build_all() {
	echo "Building release"
	if ! ./hack/make.sh "${RELEASE_BUNDLES[@]}"; then
		echo >&2
		echo >&2 'The build or tests appear to have failed.'
		echo >&2
		echo >&2 'You, as the release  maintainer, now have a couple options:'
		echo >&2 '- delay release and fix issues'
		echo >&2 '- delay release and fix issues'
		echo >&2 '- did we mention how important this is?  issues need fixing :)'
		echo >&2
		echo >&2 'As a final LAST RESORT, you (because only you, the release maintainer,'
		echo >&2 ' really knows all the hairy problems at hand with the current release'
		echo >&2 ' issues) may bypass this checking by running this script again with the'
		echo >&2 ' single argument of "--release-regardless-of-test-failure", which will skip'
		echo >&2 ' running the test suite, and will only build the binaries and packages.  Please'
		echo >&2 ' avoid using this if at all possible.'
		echo >&2
		echo >&2 'Regardless, we cannot stress enough the scarcity with which this bypass'
		echo >&2 ' should be used.  If there are release issues, we should always err on the'
		echo >&2 ' side of caution.'
		echo >&2
		exit 1
	fi
}

upload_release_build() {
	src="$1"
	dst="$2"
	latest="$3"

	echo
	echo "Uploading $src"
	echo "  to $dst"
	echo
	$AWS_CLI s3 cp --follow-symlinks --acl public-read "$src" "$dst"
	if [ "$latest" ]; then
		echo
		echo "Copying to $latest"
		echo
		$AWS_CLI s3 cp --acl public-read "$dst" "$latest"
	fi

	# get hash files too (see hash_files() in hack/make.sh)
	for hashAlgo in md5 sha256; do
		if [ -e "$src.$hashAlgo" ]; then
			echo
			echo "Uploading $src.$hashAlgo"
			echo "  to $dst.$hashAlgo"
			echo
			$AWS_CLI s3 cp --follow-symlinks --acl public-read --content-type='text/plain' "$src.$hashAlgo" "$dst.$hashAlgo"
			if [ "$latest" ]; then
				echo
				echo "Copying to $latest.$hashAlgo"
				echo
				$AWS_CLI s3 cp --acl public-read "$dst.$hashAlgo" "$latest.$hashAlgo"
			fi
		fi
	done
}

release_build() {
	echo "Releasing binaries"
	GOOS=$1
	GOARCH=$2

	binDir=bundles/$VERSION/cross/$GOOS/$GOARCH
	tgzDir=bundles/$VERSION/tgz/$GOOS/$GOARCH
	binary=docker-$VERSION
	zipExt=".tgz"
	binaryExt=""
	tgz=$binary$zipExt

	latestBase=
	if [ -z "$NOLATEST" ]; then
		latestBase=docker-latest
	fi

	# we need to map our GOOS and GOARCH to uname values
	# see https://en.wikipedia.org/wiki/Uname
	# ie, GOOS=linux -> "uname -s"=Linux

	s3Os=$GOOS
	case "$s3Os" in
		darwin)
			s3Os=Darwin
			;;
		freebsd)
			s3Os=FreeBSD
			;;
		linux)
			s3Os=Linux
			;;
		solaris)
			echo skipping solaris release
			return 0
			;;
		windows)
			# this is windows use the .zip and .exe extensions for the files.
			s3Os=Windows
			zipExt=".zip"
			binaryExt=".exe"
			tgz=$binary$zipExt
			binary+=$binaryExt
			;;
		*)
			echo >&2 "error: can't convert $s3Os to an appropriate value for 'uname -s'"
			exit 1
			;;
	esac

	s3Arch=$GOARCH
	case "$s3Arch" in
		amd64)
			s3Arch=x86_64
			;;
		386)
			s3Arch=i386
			;;
		arm)
			s3Arch=armel
			# someday, we might potentially support multiple GOARM values, in which case we might get armhf here too
			;;
		*)
			echo >&2 "error: can't convert $s3Arch to an appropriate value for 'uname -m'"
			exit 1
			;;
	esac

	s3Dir="s3://$BUCKET_PATH/builds/$s3Os/$s3Arch"
	# latest=
	latestTgz=
	if [ "$latestBase" ]; then
		# commented out since we aren't uploading binaries right now.
		# latest="$s3Dir/$latestBase$binaryExt"
		# we don't include the $binaryExt because we don't want docker.exe.zip
		latestTgz="$s3Dir/$latestBase$zipExt"
	fi

	if [ ! -f "$tgzDir/$tgz" ]; then
		echo >&2 "error: can't find $tgzDir/$tgz - was it packaged properly?"
		exit 1
	fi
	# disable binary uploads for now. Only providing tgz downloads
	# upload_release_build "$binDir/$binary" "$s3Dir/$binary" "$latest"
	upload_release_build "$tgzDir/$tgz" "$s3Dir/$tgz" "$latestTgz"
}

# Upload binaries and tgz files to S3
release_binaries() {
	[ "$(find bundles/$VERSION -path "bundles/$VERSION/cross/*/*/docker-$VERSION")" != "" ] || {
		echo >&2 './hack/make.sh must be run before release_binaries'
		exit 1
	}

	for d in bundles/$VERSION/cross/*/*; do
		GOARCH="$(basename "$d")"
		GOOS="$(basename "$(dirname "$d")")"
		release_build "$GOOS" "$GOARCH"
	done

	# TODO create redirect from builds/*/i686 to builds/*/i386

	cat <<EOF | write_to_s3 s3://$BUCKET_PATH/builds/index
# To install, run the following commands as root:
curl -fsSLO $(s3_url)/builds/Linux/x86_64/docker-$VERSION.tgz && tar --strip-components=1 -xvzf docker-$VERSION.tgz -C /usr/local/bin

# Then start docker in daemon mode:
/usr/local/bin/dockerd
EOF

	# Add redirect at /builds/info for URL-backwards-compatibility
	rm -rf /tmp/emptyfile && touch /tmp/emptyfile
	$AWS_CLI s3 cp --acl public-read --website-redirect '/builds/' --content-type='text/plain' /tmp/emptyfile "s3://$BUCKET_PATH/builds/info"

	if [ -z "$NOLATEST" ]; then
		echo "Advertising $VERSION on $BUCKET_PATH as most recent version"
		echo "$VERSION" | write_to_s3 "s3://$BUCKET_PATH/latest"
	fi
}

main() {
	[ "$SKIP_RELEASE_BUILD" = '1' ] || build_all
	setup_s3
	release_binaries
}

main

echo
echo
echo "Release complete; see $(s3_url)"
echo "Use the following text to announce the release:"
echo
echo "We have just pushed $VERSION to $(s3_url). You can download it with the following:"
echo
echo "Linux 64bit tgz: $(s3_url)/builds/Linux/x86_64/docker-$VERSION.tgz"
echo "Darwin/OSX 64bit client tgz: $(s3_url)/builds/Darwin/x86_64/docker-$VERSION.tgz"
echo "Windows 64bit zip: $(s3_url)/builds/Windows/x86_64/docker-$VERSION.zip"
echo "Windows 32bit client zip: $(s3_url)/builds/Windows/i386/docker-$VERSION.zip"
echo
