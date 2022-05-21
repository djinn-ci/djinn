#!/bin/sh

set -x

qemu-system-x86_64 -daemonize \
	-enable-kvm \
	-display none \
	-m 8192 \
	-drive file=djinn-dev,media=disk,if=virtio \
	-net nic,model=virtio \
	-net user,hostfwd=tcp:127.0.0.1:2222-:22,hostfwd=tcp:127.0.0.1:5432-:5432,hostfwd=tcp:127.0.0.1:6379-:6379 \
	-smp 2
