FROM mcr.microsoft.com/devcontainers/go:bookworm

# Install QEMU utils and capabilities(7) utils
RUN apt-get update && apt-get -y install qemu-utils libcap2-bin

# Install Cloud Hypervisor
RUN echo 'deb http://download.opensuse.org/repositories/home:/cloud-hypervisor/Debian_12/ /' | tee /etc/apt/sources.list.d/home:cloud-hypervisor.list
RUN curl -fsSL https://download.opensuse.org/repositories/home:cloud-hypervisor/Debian_12/Release.key | apt-key add -
RUN apt-get update && apt-get -y install cloud-hypervisor

# Install passt with --tap-fd support
RUN git clone --branch tap-fd https://github.com/edigaryev/passt.git
WORKDIR passt
RUN make -j `nproc` && cp passt /usr/local/bin/

# Allow /dev/kvm access for everyone
RUN chmod a+rw /dev/kvm
