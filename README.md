# ssh2lxd – SSH server for LXD containers

**ssh2lxd** is an SSH server that allows direct connections into LXD containers.
It uses LXD API in order to establish a connection with a container and create a session.

## Features

- Authentication using existing host OS SSH keys via `authorized_keys`
- SSH Agent forwarding into a container session
- Full support for PTY (terminal) mode and remote command execution
- Support for SCP and SFTP*
- Full Ansible support with fallback to SCP

> *SFTP support relies on `sftp-server` binary installed in a container (see below)

## Enterprise Features

- Authentication using any possible method (keys, passwords, external API integration, LDAP etc)
- Web browser based access to a container shell using JWT tokens
- 24/7 technical support and new feature development

## Installation

Download the latest package from **Releases** to an LXD host and install 

#### On Ubuntu / Debian

```
apt-get install -f ./ssh2lxd_1.0-0_amd64.deb
```

#### On RHEL / CentOS / AlmaLinux

```
yum install ./ssh2lxd-1.0-0.x86_64.rpm
```

#### Enable and start ssh2lxd service

```
systemctl enable ssh2lxd.service
systemctl start ssh2lxd.service
```

#### Checking logs

```
journalctl -f -u ssh2lxd.service
```

## Basic Connection

To establish an SSH connection to a container running on LXD host, run:

```
ssh -p 2222 [host-user+]container-name[+container-user]@lxd-host
```

and substitute the following

- `host-user` – active user on LXD host such as `root`
- `container-name` – running container on LXD host
- `container-user` – active user in LXD container (_optional, defaults to_ `root`)
- `lxd-host` – LXD host hostname or IP

### Examples

To connect to a container `ubuntu` running on LXD host with IP `1.2.3.4` as `root` user and authenticate
as `root` on LXD host, run:

```
ssh -p 2222 ubuntu@1.2.3.4
```

To connect to a container `ubuntu` running on LXD host with IP `1.2.3.4` as `root` user and authenticate
as `admin` on LXD host, run:

```
ssh -p 2222 admin+ubuntu@1.2.3.4
```

To connect to a container `ubuntu` running on LXD host with IP `1.2.3.4` as `ubuntu` user and authenticate
as `root` on LXD host, run:

```
ssh -p 2222 root+ubuntu+ubuntu@1.2.3.4
```

## Advanced Connection

### SSH Agent forwarding

`ssh2lxd` supports SSH Agent forwarding. To make it work in a container, it will automatically add a
proxy socket device to LXD container and remove it once SSH connection is closed.

To enable SSH agent on your local system, run:

```
eval `ssh-agent`
```

To enable SSH Agent forwarding when connecting to a container add `-A` to your `ssh` command

```
ssh -A -p 2222 ubuntu@1.2.3.4
```

### Using LXD host as SSH Proxy / Bastion

You can access an LXD container by using LXD host's SSH server as a Proxy / Bastion.
The easiest way is to add additional configuration to your `~/.ssh/config`

```
Host lxd1
  Hostname localhost
  Port 2222
  ProxyJump lxd-host

Host lxd-host
  Hostname 1.2.3.4
  User root
```

Now to connect to `ubuntu` container as `root`, run:

```
ssh ubuntu@lxd1
```

> Using this method has additional security benefits and port 2222 is not exposed to the public 

### SFTP Connection

In order to enable full SFTP support on an LXD container it needs `sftp-server` binary installed. And it doesn't require
`sshd` service to run in a container.

#### Ubuntu / Debian containers

```
apt-get update
apt-get install openssh-sftp-server
```

#### CentOS / Fedora containers

```
yum install openssh-server
```

#### Alpine Linux containers

```
apk update
apk add openssh-sftp-server
```

### Ansible

Running Ansible commands and playbooks directly on LXD containers is fully support with or without `sftp-server` binary
in a container. Ansible falls back to SCP mode when SFTP is not available.

#### Examples

```
ansible.cfg:

[defaults]
host_key_checking = False
remote_tmp = /tmp/.ansible-${USER}
```

```
inventory:

# Direct connection to port 2222
[lxd1]
container-a ansible_user=root+c1 ansible_host=1.2.3.4 ansible_port=2222
container-b ansible_user=root+u1+ubuntu ansible_host=1.2.3.4 ansible_port=2222 become=yes

# Connection using ProxyJump configured in ssh config 
[lxd2]
container-c ansible_user=root+c1 ansible_host=lxd1
container-d ansible_user=root+u1+ubuntu ansible_host=lxd1 become=yes
```

```
playbook.yml:

---
- hosts: lxd1,lxd2
  become: no
  become_method: sudo

  tasks:
    - command: env
    - command: ip addr
```


## Configuration Options

By default `ssh2lxd` will listen on port `2222` and allow authentication for `root` and users who belong to the groups
`adm,lxd` on Ubuntu / Debian LXD host and `wheel,lxd` on RHEL LXD host.

To add a user to one of those groups run as root `usermod -aG lxd your-host-user`

To run `ssh2lxd` with custom configuration options you can edit `/etc/default/ssh2lxd` on Ubuntu / Debian or
`/etc/sysconfig/ssh2lxd` on RHEL systems. The following options can be added to `ARGS=`

```
-d, --debug                enable debug log
-g, --groups string        list of groups members of which allowed to connect (default "adm,lxd")
    --healthcheck string   enable LXD health check every X minutes, e.g. "5m"
-l, --listen string        listen on :2222 or 127.0.0.1:2222 (default ":2222")
    --noauth               disable SSH authentication completely
-s, --socket string        LXD socket or use LXD_SOCKET (default "/var/snap/lxd/common/lxd/unix.socket")
```

For example, to enable debug log and listen on localhost change the line to `ARGS=-d -l 127.0.0.1:2222`

### Firewall

If you have firewall enabled on your LXD host, you may need to allow connections to port `2222`

On Ubuntu / Debian

```
ufw allow 2222/tcp
ufw reload
```

On RHEL / CentOS / AlmaLinux

```
firewall-cmd --permanent --add-port=2222/tcp
firewall-cmd --reload
```

## Support

Community support is available through **GitHub Issues**.
