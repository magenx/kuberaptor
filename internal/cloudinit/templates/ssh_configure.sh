#!/bin/bash

# Disable ssh socket for config consistency
if systemctl is-enabled ssh.socket > /dev/null 2>&1
then
  systemctl disable --now ssh.socket
  systemctl enable --now ssh.service
fi

# Update ssh port from config
cat > /etc/ssh/sshd_config.d/20-kuberaptor-custom-port.conf << EOF
Port {{ .ssh_port }}
EOF

# SSH optimization and security overrides
cat > /etc/ssh/sshd_config.d/10-kuberaptor-security.conf << EOF
LoginGraceTime 30
MaxAuthTries 3
X11Forwarding no
PrintLastLog no
UseDNS no
PrintMotd no
TCPKeepAlive yes
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin prohibit-password
ClientAliveInterval 120
ClientAliveCountMax 2
EOF

systemctl restart sshd
