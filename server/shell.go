package server

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"ssh2lxd/lxd"
	"ssh2lxd/lxd/device"
	"ssh2lxd/util/shlex"
	"ssh2lxd/util/ssh"

	"github.com/creack/pty"
	log "github.com/sirupsen/logrus"
)

func shellHandler(s ssh.Session) {
	env := make(map[string]string)

	user, instance, instanceUser := parseUser(s.User())
	log.Debugf("shell: connecting %s to %s as %s", user, instance, instanceUser)

	if user == "root" && instance == "%shell" {
		lxcShell(s)
		return
	}

	server, err := lxd.Connect(config.LxdSocket)
	if err != nil {
		log.Errorln(err.Error())
		s.Exit(255)
		return
	}
	defer server.Disconnect()

	var iu *lxd.InstanceUser
	if instanceUser != "" {
		iu = lxd.GetInstanceUser(server, instance, instanceUser)
	}

	if iu == nil {
		io.WriteString(s, "not found user or instance\n")
		s.Exit(1)
		return
	}

	if ssh.AgentRequested(s) {
		l, err := ssh.NewAgentListener()
		if err != nil {
			log.Errorln(err.Error())
		} else {
			defer l.Close()
			go ssh.ForwardAgentConnections(l, s)

			d := &device.ProxySocket{
				Server:   &server,
				Instance: instance,
				Source:   l.Addr().String(),
				Uid:      iu.Uid,
				Gid:      iu.Gid,
				Mode:     "0660",
			}

			if socket, err := d.AddProxySocket(); err == nil {
				env["SSH_AUTH_SOCK"] = socket
				defer d.RemoveProxySocket()
			} else {
				log.Errorln(err.Error())
			}
		}
	}

	ptyReq, winCh, isPty := s.Pty()

	for _, v := range s.Environ() {
		k := strings.Split(v, "=")
		env[k[0]] = k[1]
	}
	if ptyReq.Term != "" {
		env["TERM"] = ptyReq.Term
	} else {
		env["TERM"] = "xterm-256color"
	}

	env["USER"] = iu.User
	env["HOME"] = iu.Dir

	var cmd string
	if s.RawCommand() == "" {
		cmd = iu.Shell
	} else {
		cmd = s.RawCommand()
	}

	log.Debugf("Cmd: %s", cmd)
	log.Debugf("Pty: %v", isPty)
	log.Debugf("Env: %v", env)

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

	windowChannel := make(lxd.WindowChannel)
	go func() {
		for win := range winCh {
			windowChannel <- lxd.Window{Width: win.Width, Height: win.Height}
		}
	}()

	ie := &lxd.InstanceExec{
		Server:   &server,
		Instance: instance,
		Cmd:      cmd,
		Env:      env,
		IsPty:    isPty,
		Window:   lxd.Window(ptyReq.Window),
		WinCh:    windowChannel,
		Stdin:    stdin,
		Stdout:   s,
		Stderr:   stderr,
		User:     iu.Uid,
		Group:    iu.Gid,
		Cwd:      iu.Dir,
	}

	ret, err := ie.Exec()
	if err != nil {
		log.Debugln("shell: connection failed")
	}

	s.Exit(ret)
}

func lxcShell(s ssh.Session) {
	cmdString := `-c 'while true; do read -r -p "
Type lxc command:
> lxc " a; lxc $a; done'`

	args, _ := shlex.Split(cmdString, true)
	cmd := exec.Command("bash", args...)

	ptyReq, winCh, isPty := s.Pty()
	if isPty {
		cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
		cmd.Env = append(cmd.Env, "PATH=/bin:/usr/bin:/var/lib/snapd/snap/bin:/usr/local/bin")
		cmd.Env = append(cmd.Env, fmt.Sprintf("LXD_SOCKET=%s", config.LxdSocket))
		//for _, e := range s.Environ() {
		//	cmd.Env = append(cmd.Env, e)
		//}
		f, err := pty.Start(cmd)
		if err != nil {
			log.Errorln(err.Error())
			io.WriteString(s, "Couldn't allocate PTY\n")
			s.Exit(-1)
		}
		io.WriteString(s, `
lxc shell emulator. Use Ctrl+c to exit

Hit Enter or type 'help' for help
`)
		go func() {
			for win := range winCh {
				setWinsize(f, win.Width, win.Height)
			}
		}()
		go func() {
			io.Copy(f, s) // stdin
		}()
		io.Copy(s, f) // stdout
		cmd.Wait()
	} else {
		io.WriteString(s, "No PTY requested.\n")
		s.Exit(1)
	}
}

func setWinsize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}
