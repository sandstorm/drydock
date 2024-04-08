# Contributing

We'd love to have pull requests or bug reports :-) In case we do not react timely,
do not hesitate to get in touch with us e.g. via [kontakt@sandstorm.de](mailto:kontakt@sandstorm.de),
as it might happen that a pull request slips through.

## Developing

Simply have a modern Go version installed; check out the project somewhere (NOT in $GOPATH, as we use Go Modules),
and then run `./dev.sh build`.

### Developing eBPF on Mac OS

```bash
limactl create --name=drydock ./util/lima-dev-vm/lima-dev-vm.template.yml --tty=false
limactl start drydock

limactl shell drydock


# removing everything to start from scratch
limactl stop drydock
limactl delete drydock

```

apt install clang llvm libbpf-dev golang
sudo apt install linux-headers-$(uname -r)
sudo ln -sf /usr/include/asm-generic/ /usr/include/asm

cd /drydock/cmd/docker-net-connect/ebpf-xdp/
go generate
go build
sudo ./ebpf-xdp

## Releasing

```bash

git push
git tag v3.x.y
git push origin v3.x.y

goreleaser --rm-dist
```
