#!/bin/bash

echo "Building and installing ping_graph"

cd ./server
git_commit=$(git rev-parse HEAD)
cp ./version.go temp_version.go
sed -i "4s/.\{16\}/&$git_commit/" ./temp_version.go
go build -o ping_graph ./server.go ./websocket.go ./temp_version.go
rm ./temp_version.go
cd ..
echo "- Build complete"

if systemctl is-active --quiet ping_graph; then
    systemctl stop ping_graph
    echo "- Stopped currently running service"
else
    echo "- Service not running"
fi

rm -rf /srv/ping_graph
mkdir -p /srv/ping_graph/static
cp -r ./static/ /srv/ping_graph/

install_dir="/usr/local/bin"
mkdir -p $install_dir
mv -f ./server/ping_graph $install_dir
sudo setcap cap_net_raw=+ep $install_dir/ping_graph
printf -- "- Installed to %s\n" $install_dir

cp -f ./ping_graph.service /etc/systemd/system/ping_graph.service
# Required to use the updated .service file.
sudo systemctl daemon-reload
systemctl enable ping_graph
systemctl start ping_graph
echo "- Registered and started service"
