# Index

* [Prebuilt Binary](#prebuilt-binary)
* [From Source](#from-source)

# Prerequisites

Make sure that your user has the `/dev/kvm` access. On most distributions, this can be accomplished by adding the current user to the `kvm` group:

```shell
sudo gpasswd -a $USER kvm
```

Once added to a group, you will need to re-login for the changes to take effect.

## Installation

## Prebuilt Binary

Check the [releases page](https://github.com/openai/vetu/releases) for a pre-built `vetu` binary for your platform.

Here's a one-liner for Linux to download the latest release:

```bash
arch="$(uname -m)"
case "$arch" in
  x86_64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
esac
curl -L -o vetu "https://github.com/openai/vetu/releases/latest/download/vetu-linux-$arch"
sudo install -m 0755 vetu /usr/bin/vetu
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/bin/vetu
```

## From Source

If you have [Golang](https://golang.org/) 1.21 or newer installed, you can run:

```
go install github.com/cirruslabs/vetu/cmd/vetu@latest
```

This will build and place the `vetu` binary in `$GOPATH/bin`.

Vetu binary also needs some capabilities assigned to it:

```shell
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip $GOPATH/bin/vetu
```

To be able to run `vetu` command from anywhere, make sure the `$GOPATH/bin` directory is added to your `PATH`
environment variable (see [article in the Go wiki](https://github.com/golang/go/wiki/SettingGOPATH) for more details).
