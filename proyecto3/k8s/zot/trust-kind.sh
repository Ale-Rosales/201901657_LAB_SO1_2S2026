#!/bin/bash
set -e
NODE=munk8s-control-plane
docker exec $NODE mkdir -p "/etc/containerd/certs.d/zot:5000"
docker cp ~/zot/certs/zot.crt "$NODE:/etc/containerd/certs.d/zot:5000/ca.crt"
cat > /tmp/hosts.toml <<TOML
server = "https://zot:5000"

[host."https://zot:5000"]
  capabilities = ["pull", "resolve"]
  ca = "/etc/containerd/certs.d/zot:5000/ca.crt"
TOML
docker cp /tmp/hosts.toml "$NODE:/etc/containerd/certs.d/zot:5000/hosts.toml"
docker exec $NODE bash -c '
grep -q "config_path" /etc/containerd/config.toml || cat >> /etc/containerd/config.toml <<EOF

[plugins."io.containerd.grpc.v1.cri".registry]
  config_path = "/etc/containerd/certs.d"
EOF
systemctl restart containerd'
