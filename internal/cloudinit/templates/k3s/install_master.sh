#!/bin/bash
# K3s installation script for master nodes
# All masters use the same command with --cluster-init flag, enabling parallel installation
# and automatic etcd cluster formation

curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION='{{ .K3sVersion }}' K3S_TOKEN='{{ .K3sToken }}' sh -s - server {{ .BaseArgs }}
