FROM ubuntu:mantic

RUN apt-get update
RUN apt-get install -y git build-essential libelf-dev llvm libbfd-dev libcap-dev
RUN git clone --recurse-submodules https://github.com/libbpf/bpftool.git
RUN cd bpftool/src; make install


# /usr/local/sbin/bpftool btf dump file /sys/kernel/btf/vmlinux format c

