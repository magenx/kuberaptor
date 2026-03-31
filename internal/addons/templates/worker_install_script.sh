#!/bin/bash
set -o pipefail

touch /etc/initialized

HOSTNAME=$(hostname -f)
PUBLIC_IP=$(hostname -I | awk '{print $1}')

# Network configuration
if [ "{{ .PrivateNetworkEnabled }}" = "true" ]; then
  echo "Using Hetzner private network" >/var/log/kuberaptor.log
  SUBNET="{{ .PrivateNetworkSubnet }}"

  # Wait for private network interface to be available
  MAX_ATTEMPTS=30
  DELAY=10

  for i in $(seq 1 $MAX_ATTEMPTS); do
    # Simplified network interface detection
    NETWORK_INTERFACE=$(
      ip -o link show |
        awk -F': ' '/mtu (1450|1280)/ {print $2}' |
        grep -Ev 'cilium|br|flannel|docker|veth' |
        head -n1
    )

    if [ -n "$NETWORK_INTERFACE" ]; then
      echo "Private network interface $NETWORK_INTERFACE found" 2>&1 | tee -a /var/log/kuberaptor.log
      break
    fi

    echo "Waiting for private network interface in subnet $SUBNET... (Attempt $i/$MAX_ATTEMPTS)" 2>&1 | tee -a /var/log/kuberaptor.log
    sleep $DELAY
  done

  # Check if we found the interface
  if [ -z "$NETWORK_INTERFACE" ]; then
    echo "ERROR: Timeout waiting for private network interface in subnet $SUBNET" 2>&1 | tee -a /var/log/kuberaptor.log
    exit 1
  fi

  # Get private IP address
  PRIVATE_IP=$(
    ip -4 -o addr show dev "$NETWORK_INTERFACE" |
      awk '{print $4}' |
      cut -d'/' -f1 |
      head -n1
  )

  # Verify we got a private IP
  if [ -z "$PRIVATE_IP" ]; then
    echo "ERROR: Could not determine private IP address for interface $NETWORK_INTERFACE" 2>&1 | tee -a /var/log/kuberaptor.log
    exit 1
  fi

  echo "Private network IP: $PRIVATE_IP" 2>&1 | tee -a /var/log/kuberaptor.log
  FLANNEL_SETTINGS="--flannel-iface=$NETWORK_INTERFACE"
else
  echo "Using public network" >/var/log/kuberaptor.log
  PRIVATE_IP="${PUBLIC_IP}"
  FLANNEL_SETTINGS=""
fi

# Create k3s directories
mkdir -p /etc/rancher/k3s

# Create registries.yaml
cat >/etc/rancher/k3s/registries.yaml <<EOF
mirrors:
  "*":
EOF

# Get instance ID for public network
KUBELET_INSTANCE_ID=""
if [ "{{ .PrivateNetworkEnabled }}" = "false" ]; then
  INSTANCE_ID=$(curl -s http://169.254.169.254/hetzner/v1/metadata/instance-id)
  if [ -n "$INSTANCE_ID" ]; then
    KUBELET_INSTANCE_ID="--kubelet-arg=provider-id=hcloud://$INSTANCE_ID"
  else
    echo "WARNING: Could not retrieve instance ID" 2>&1 | tee -a /var/log/kuberaptor.log
  fi
fi

# Install k3s worker
echo "Installing k3s worker..." 2>&1 | tee -a /var/log/kuberaptor.log

# Write token to a temporary file to avoid shell escaping issues
echo -n "{{ .K3sToken }}" > /tmp/k3s-token
chmod 600 /tmp/k3s-token

curl -sfL https://get.k3s.io | \
  K3S_TOKEN="$(cat /tmp/k3s-token)" \
  INSTALL_K3S_VERSION="{{ .K3sVersion }}" \
  K3S_URL=https://{{ .MasterIP }}:6443 \
  INSTALL_K3S_EXEC="agent" \
  sh -s - \
    --node-name=$HOSTNAME \
    {{ .KubeletArgs }} {{ .LabelsAndTaints }} \
    --node-ip=$PRIVATE_IP \
    --node-external-ip=$PUBLIC_IP \
    $KUBELET_INSTANCE_ID \
    $FLANNEL_SETTINGS 2>&1 | tee -a /var/log/kuberaptor.log

# Store the exit code
INSTALL_EXIT_CODE=$?

# Clean up token file
rm -f /tmp/k3s-token

# Check if installation was successful
if [ $INSTALL_EXIT_CODE -ne 0 ]; then
  echo "ERROR: k3s worker installation failed with exit code $INSTALL_EXIT_CODE" 2>&1 | tee -a /var/log/kuberaptor.log
  exit 1
fi

echo "k3s worker installation completed successfully" 2>&1 | tee -a /var/log/kuberaptor.log
echo true >/etc/initialized
