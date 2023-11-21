#!/bin/sh

set -e

/sbin/setcap cap_net_raw,cap_net_admin+eip /usr/bin/vetu
