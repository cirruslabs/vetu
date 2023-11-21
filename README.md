# **vetu** - Virtualization that is Easy To Use

_vetu_ is virtualization toolset to effortlessly run [Cloud Hypervisor](https://www.cloudhypervisor.org/)-backed virtual machines on Linux hosts.

We say effortlessly, because the existing virtualization solutions like the traditional [QEMU](https://www.qemu.org/) and the new-wave [Firecracker](https://firecracker-microvm.github.io/) and Cloud Hypervisor provide lots of options and require users to essentially build a tooling on top of them to be able to simply run a basic VM.

Vetu builds on the success of [Tart](https://tart.run/) and abstracts all these peculiarities and makes the virtualization as easy as running containers.

Here are just some of the cool features that Vetu inherited from Tart:

* Ability to easily distribute VM images by integrating with OCI-compatible container registries. Push and pull virtual machines like they are containers.
* Effortless SSH'ing into VMs (see [Usage](#usage) for an example)
* [Cirrus CLI](https://github.com/cirruslabs/cirrus-cli) integration

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

## Networking options

### Default (NAT)

The default NAT networking is powered by the [gVisor's TCP/IP stack](https://gvisor.dev/docs/user_guide/networking/) and has an advantage of requiring no configuration on the user's end.

The main disadvantage is the reduced speed, because all the routing and processing is also done in the software (in the Vetu process), in addition to the kernel.

However, this choice is still fast enough to run most of the tasks, for example, provisioning a Linux distro and installing the packages.

### Bridged

Bridged networking can be enabling by specifying `--net-bridged=BRIDGE_INTERFACE_NAME` argument to `vetu run` and has an advantage of being fast, because all the processing and routing is done in the kernel.

The main disadvantage is that this choice requires the system administrator to properly configure the bridge interface, IP forwarding, DHCP server (if required by the VM) and the packet filter to provide adequate network isolation.

## FAQ

### VM location on disk

Vetu stores all it's files in `~/.vetu/` directory. Local images that you can run are stored in `~/.vetu/vms/`. Remote images are pulled into `~/.vetu/cache/OCIs/`.
