#!/bin/sh

set -e

/sbin/setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/bin/vetu
