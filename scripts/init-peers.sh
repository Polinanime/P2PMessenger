#!/bin/sh
mkdir -p src/.config
cat > src/.config/peers.txt << EOF
172.20.0.2:1235
172.21.0.2:1236
172.21.0.3:1237
EOF
