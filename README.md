# wg-tun2gvisor
A userspace VPN server (Wireguard-Go) which desn't need TUN/TAP device. Created for prototype / demo purposes.

It forwards TCP and UDP traffic from wireguard tunnel to network using gvisor's userspace implementation of network stack.
Does not require any extra privileges.
