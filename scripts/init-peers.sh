#!/bin/sh
mkdir -p .config
cat > .config/peers.txt << EOF
172.20.0.2:1235
172.20.0.3:1236
172.20.0.4:1237
EOF
