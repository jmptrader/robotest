#!/bin/bash
#
# File passed to VM at creation time
#
set -euo pipefail

function create_version_script {
  local usage="$FUNCNAME script_path"
  local script_path=${1:?$usage}
  cat >$script_path <<EOF
#!/bin/bash
set -eo pipefail; [[ \$TRACE ]] && set -x

function die {
    echo "ERROR: \$*" >&2
    exit 1
}

# specifies the first version capable of outputting status in JSON
readonly base_version=4.44.0

# ver1 >= ver2
function version_gte {
  local usage="\$FUNCNAME ver1 ver2"
  local ver1=\${1:?\$usage}
  if [ "\$ver1" != "\$base_version" ]; then
    test "\$(printf '%s\n' "\$@" | sort -V | head -n 1)" != "\$ver1";
  fi
}

readonly help="gravity_status <gravity binary> <arg>..."
readonly gravity=\${1:?\$help}
shift
readonly bin_version=\$(\$gravity version | head -1 | sed -e 's/version\:\s\+//' | egrep -o '^([0-9]+)\.([0-9]+)\.([0-9]+)')
if version_gte \$bin_version \$base_version; then
  \$gravity status --output=json "\$@"
else
  \$gravity status "\$@"
fi
EOF
  chmod +x $script_path
}

touch /var/lib/bootstrap_started

# disable Hyper-V time sync
echo 2dd1ce17-079e-403c-b352-a1921ee207ee > /sys/bus/vmbus/drivers/hv_util/unbind

apt update 
apt install -y chrony lvm2 curl wget thin-provisioning-tools
curl https://bootstrap.pypa.io/get-pip.py | python -
pip install --upgrade awscli

mkfs.ext4 -F /dev/sdc
echo -e '/dev/sdc\t/var/lib/gravity/planet/etcd\text4\tdefaults\t0\t2' >> /etc/fstab

mkdir -p /var/lib/gravity/planet/etcd /var/lib/data
mount /var/lib/gravity/planet/etcd

chown -R 1000:1000 /var/lib/gravity /var/lib/data /var/lib/gravity/planet/etcd
sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers

create_version_script /tmp/gravity_status.sh

# robotest might SSH before bootstrap script is complete (and will fail)
touch /var/lib/bootstrap_complete
