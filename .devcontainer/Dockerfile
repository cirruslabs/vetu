FROM mcr.microsoft.com/devcontainers/go:bookworm

# Install QEMU, capabilities(7) and other useful utilities
RUN apt-get update && apt-get -y install qemu-utils libcap2-bin iputils-ping genisoimage

# Install Cloud Hypervisor
RUN echo 'deb http://download.opensuse.org/repositories/home:/cloud-hypervisor/Debian_12/ /' | tee /etc/apt/sources.list.d/home:cloud-hypervisor.list
RUN curl -fsSL https://download.opensuse.org/repositories/home:cloud-hypervisor/Debian_12/Release.key | apt-key add -
RUN apt-get update && apt-get -y install cloud-hypervisor edk2-cloud-hypervisor
