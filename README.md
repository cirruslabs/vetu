# vetu

_vetu_ is virtualization toolset to effortlessly run [Cloud Hypervisor](https://www.cloudhypervisor.org/)-backed virtual machines on Linux hosts.

## Installation

* [Debian-based distributions](INSTALL.md#debian-based-distributions) (Debian, Ubuntu, etc.)
* [RPM-based distributions](INSTALL.md#rpm-based-distributions) (Fedora, CentOS, etc.)
* [Prebuilt Binary](INSTALL.md#prebuilt-binary)
* [From Source](INSTALL.md#from-source)

## Usage

Try running a Vetu VM on your Linux machine with `arm64` processor:

```shell
vetu clone ghcr.io/cirruslabs/ubuntu:latest ubuntu
vetu run ubuntu
```

The default username is `admin` and password is `admin`. The machine is only reachable from the localhost with the default configuration, and you can connect to it over SSH using the following command:

```shell
ssh admin@$(vetu ip ubuntu)
```

## FAQ

### VM location on disk

Vetu stores all it's files in `~/.vetu/` directory. Local images that you can run are stored in `~/.vetu/vms/`. Remote images are pulled into `~/.vetu/cache/OCIs/`.
