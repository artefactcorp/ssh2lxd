package lxd

import (
	"fmt"
	"strconv"
	"strings"

	"ssh2lxd/util/buffer"

	"github.com/lxc/lxd/client"
	log "github.com/sirupsen/logrus"
)

type InstanceUser struct {
	Instance string
	User     string
	Uid      int
	Gid      int
	Dir      string
	Shell    string
	Ent      string
}

func GetInstanceUser(server lxd.InstanceServer, instance string, user string) *InstanceUser {
	cmd := fmt.Sprintf("getent passwd %s", user)
	stdout := buffer.NewOutputBuffer()
	stderr := buffer.NewOutputBuffer()

	ie := &InstanceExec{
		Server:   &server,
		Instance: instance,
		Cmd:      cmd,
		Stdout:   stdout,
		Stderr:   stderr,
	}

	ret, _ := ie.Exec()

	if ret == 0 && len(stdout.Lines()) > 0 {
		ent := strings.Split(stdout.Lines()[0], ":")
		user = ent[0]
		uid, _ := strconv.Atoi(ent[2])
		gid, _ := strconv.Atoi(ent[3])
		dir := ent[5]
		shell := ent[6]
		iu := &InstanceUser{
			Instance: instance,
			User:     user,
			Uid:      uid,
			Gid:      gid,
			Dir:      dir,
			Shell:    shell,
			Ent:      stdout.Lines()[0],
		}

		return iu
	}

	log.Debugf("couldn't find user %s or instance %s", user, instance)

	return nil
}
