# Create bridge
sudo ip link add name br0 type bridge
sudo ip addr add 172.16.0.1/24 dev br0
sudo ip link set br0 up

# Enable IP forwarding + NAT for internet access (once)
sudo sysctl -w net.ipv4.ip_forward=1
sudo iptables -t nat -A POSTROUTING -s 172.16.0.0/24 ! -o br0 -j MASQUERADE

sudo iptables -C FORWARD -i br0 -j ACCEPT || sudo iptables -A FORWARD -i br0 -j ACCEPT
sudo iptables -C FORWARD -o br0 -m state --state RELATED,ESTABLISHED -j ACCEPT \
  || sudo iptables -A FORWARD -o br0 -m state --state RELATED,ESTABLISHED -j ACCEPT