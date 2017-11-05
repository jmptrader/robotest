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

# ver1 > ver2
function version_gt {
  local usage="\$FUNCNAME ver1 ver2"
  local ver1=\${1:?\$usage}
  test "\$(printf '%s\n' "\$@" | sort -V | head -n 1)" != "\$ver1";
}

readonly help="gravity_status <gravity binary> <arg>..."
readonly gravity=\${1:?\$help}
shift
readonly bin_version=\$(\$gravity version | head -1 | sed -e 's/version\:\s\+//' | egrep -o '^([0-9]+)\.([0-9]+)\.([0-9]+)')
if version_gt \$bin_version \$base_version; then
  args=()
  for param in "\$@"; do
    # ignore quiet parameter as versions following base_version properly handle quiet mode - e.g. suppress output
    [[ ("\$param" != "--quiet") && ("\$param" != "-q") ]] && args+=("\$param")
  done
  \$gravity status --output=json "\${args[@]}"
else
  \$gravity status "\$@"
fi
EOF
  chmod +x $script_path
}

apt update 
apt install -y python-pip lvm2 curl wget
pip install --upgrade awscli

mkfs.ext4 /dev/xvdc
echo -e '/dev/xvdc\t/var/lib/gravity/planet/etcd\text4\tdefaults\t0\t2' >> /etc/fstab

mkdir -p /var/lib/gravity/planet/etcd /var/lib/data
mount /var/lib/gravity/planet/etcd

chown -R 1000:1000 /var/lib/gravity /var/lib/data /var/lib/gravity/planet/etcd
sed -i.bak 's/Defaults    requiretty/#Defaults    requiretty/g' /etc/sudoers

umount /dev/xvdb || true
wipefs -a /dev/xvdb || true

create_version_script /tmp/gravity_status.sh

# robotest might SSH before bootstrap script is complete (and will fail)
touch /var/lib/bootstrap_complete
