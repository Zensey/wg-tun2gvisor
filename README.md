# wg-tun2gvisor
A userspace VPN using Wireguard-Go and gVisor. Created for prototype / demo purposes.

It forwards TCP and UDP traffic from wireguard tunnel to network using gvisor userspace network stack.
Does not require any extra privileges.
