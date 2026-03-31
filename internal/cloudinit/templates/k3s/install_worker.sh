#!/bin/bash
# K3s installation script for worker nodes
# This script is executed on worker nodes to join the k3s cluster as agents

curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION='{{ .K3sVersion }}' K3S_TOKEN='{{ .K3sToken }}' K3S_URL='{{ .K3sURL }}' sh -s - agent{{ .BaseArgs }}
