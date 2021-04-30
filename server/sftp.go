package server

import (
	"fmt"
	"io"
	"strings"

	"ssh2lxd/lxd"
	"ssh2lxd/util/ssh"

	log "github.com/sirupsen/logrus"
)

var sftpServerBins = map[string]string{
	"alpinelinux": "/usr/lib/ssh/sftp-server",
	"centos":      "/usr/libexec/openssh/sftp-server",
	"debian":      "/usr/lib/openssh/sftp-server",
	"fedora":      "/usr/libexec/openssh/sftp-server",
	"ubuntu":      "/usr/lib/openssh/sftp-server",
}

func sftpSubsystemHandler(s ssh.Session) {
	user, instance, instanceUser := parseUser(s.User())
	log.Debugf("sftp: connecting %s to %s as %s\n", user, instance, instanceUser)

	server, err := lxd.Connect(config.LxdSocket)
	if err != nil {
		log.Errorln(err.Error())
		s.Exit(2)
		return
	}
	defer server.Disconnect()

	meta, _, _ := lxd.GetInstanceMeta(server, instance)

	os := strings.ToLower(meta.Properties["os"])

	log.Debugln(meta)

	sftpServerBin, ok := sftpServerBins[os]
	if !ok {
		log.Errorf("sftp: unknown sftp-server binary for %s", os)
		io.WriteString(s, fmt.Sprintf("unknown OS %s\n", os))
		s.Exit(2)
		return
	}

	var iu *lxd.InstanceUser
	if instanceUser != "" {
		iu = lxd.GetInstanceUser(server, instance, instanceUser)
	}

	if iu == nil {
		io.WriteString(s, "not found user or instance\n")
		s.Exit(1)
		return
	}

	log.Debugf("sftp: found instance user %s [%d %d]", iu.User, iu.Uid, iu.Gid)

	stdin, inWrite := io.Pipe()
	errRead, stderr := io.Pipe()

	go func(s ssh.Session, w io.WriteCloser) {
		defer w.Close()
		io.Copy(w, s)
	}(s, inWrite)

	go func(s ssh.Session, e io.ReadCloser) {
		defer e.Close()
		io.Copy(s.Stderr(), e)
	}(s, errRead)

	cmd := fmt.Sprintf("%s -e -d %s", sftpServerBin, iu.Dir)

	ie := &lxd.InstanceExec{
		Server:   &server,
		Instance: instance,
		Cmd:      cmd,
		Stdin:    stdin,
		Stdout:   s,
		Stderr:   stderr,
		User:     iu.Uid,
		Group:    iu.Gid,
	}

	ret, err := ie.Exec()
	if err != nil {
		log.Debugln("sftp: connection failed")
	}

	s.Exit(ret)
}
