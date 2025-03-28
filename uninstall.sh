#!/bin/bash

set -e

rm -rf /srv/ping_graph

systemctl stop ping_graph
systemctl disable ping_graph
rm -f /usr/local/bin/ping_graph
rm -f /etc/systemd/system/ping_graph.service
systemctl daemon-reload

echo "Uninstalled ping_graph"
