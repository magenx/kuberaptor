#!/bin/bash
# Internet connectivity test script
# Tests connectivity to the k3s installation endpoint
# Returns 'connected' if successful

curl -s --connect-timeout 5 --max-time 10 https://get.k3s.io >/dev/null 2>&1 && echo 'connected'
