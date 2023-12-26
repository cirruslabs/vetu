# Index

* [Debian-based distributions](#debian-based-distributions) (Debian, Ubuntu, etc.)
* [RPM-based distributions](#rpm-based-distributions) (Fedora, CentOS, etc.)
* [Prebuilt Binary](#prebuilt-binary)
* [From Source](#from-source)

# Prerequisites

Make sure that your user has the `/dev/kvm` access. On most distributions, this can be accomplished by adding the current user to the `kvm` group:

```shell
sudo gpasswd -a $USER kvm
```

Once added to a group, you will need to re-login for the changes to take effect.

## Installation

## Debian-based distributions

First, make sure that you've installed the APT transport for downloading packages via HTTPS and common X.509 certificates:

```shell
sudo apt-get update && sudo apt-get -y install apt-transport-https ca-certificates
```

Then, add the Cirrus Labs repository:

```shell
echo "deb [trusted=yes] https://apt.fury.io/cirruslabs/ /" | sudo tee /etc/apt/sources.list.d/cirruslabs.list
```

Now you can update the package index files and install the Vetu:

```shell
sudo apt-get update && sudo apt-get -y install vetu
```

## RPM-based distributions

First, create a `/etc/yum.repos.d/cirruslabs.repo` file with the following contents:

```
[cirruslabs]
name=Cirrus Labs Repo
baseurl=https://yum.fury.io/cirruslabs/
enabled=1
gpgcheck=0
```

Now you can install the Vetu:

```shell
sudo yum -y install vetu
```

## Prebuilt Binary

Check the [releases page](https://github.com/cirruslabs/vetu/releases) for a pre-built `vetu` binary for your platform.

Here's a one-liner for Linux to download the latest release:

```bash
curl -L -o vetu https://github.com/cirruslabs/vetu/releases/latest/download/vetu-linux-$(uname -m) && sudo mv vetu /usr/bin/vetu && sudo chmod +x /usr/bin/vetu && sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/bin/vetu
```

## From Source

If you have [Golang](https://golang.org/) 1.21 or newer installed, you can run:

```
go install github.com/cirruslabs/vetu/...@latest
```

This will build and place the `vetu` binary in `$GOPATH/bin`.

Vetu binary also needs some capabilities assigned to it:

```shell
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip $GOPATH/bin/vetu
```

To be able to run `vetu` command from anywhere, make sure the `$GOPATH/bin` directory is added to your `PATH`
environment variable (see [article in the Go wiki](https://github.com/golang/go/wiki/SettingGOPATH) for more details).
