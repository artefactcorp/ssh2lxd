package server

import (
	"fmt"
	"time"

	"ssh2lxd/lxd"
	"ssh2lxd/util/ssh"

	log "github.com/sirupsen/logrus"
	"gopkg.in/robfig/cron.v2"
)

type Config struct {
	IdleTimeout   time.Duration
	Debug         bool
	Listen        string
	Socket        string
	Noauth        bool
	Groups        string
	HealthCheck   string
	AllowedGroups []string
	LxdSocket     string

	lxdInfo map[string]interface{}
}

var config *Config

func Run(c *Config) {
	config = c

	if err := checkLxd(); err != nil {
		log.Fatal(err.Error())
	}

	if len(config.HealthCheck) > 0 {
		enableHealthCheck()
	}

	var authHandler ssh.PublicKeyHandler
	if config.Noauth {
		authHandler = nil
	} else {
		authHandler = keyAuthHandler
	}

	if len(config.AllowedGroups) > 0 {
		config.AllowedGroups = append([]string{"0"}, getGroupIds(c.AllowedGroups)...)
	}

	var defaultSubsystemHandler ssh.SubsystemHandler = defaultSubsystemHandler
	var sftpSubsystemHandler ssh.SubsystemHandler = sftpSubsystemHandler

	server := &ssh.Server{
		Addr:             config.Listen,
		IdleTimeout:      config.IdleTimeout,
		Version:          "LXD",
		PublicKeyHandler: authHandler,
		Handler:          shellHandler,
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"default": defaultSubsystemHandler,
			"sftp":    sftpSubsystemHandler,
		},
	}

	log.Fatal(server.ListenAndServe())
}

func GetConfig() Config {
	return *config
}

func enableHealthCheck() {
	c := cron.New()
	c.AddFunc(fmt.Sprintf("@every %s", config.HealthCheck), checkHealth)
	c.Start()
}

func checkLxd() error {
	s, err := lxd.Connect(config.LxdSocket)
	if err != nil {
		return err
	}

	info := lxd.GetConnectionInfo(s)
	config.lxdInfo = info
	log.Debugln(info)

	s.Disconnect()
	return nil
}

func checkHealth() {
	err := checkLxd()
	if err != nil {
		log.Errorln("Health check failed", err.Error())
	}
}

func defaultSubsystemHandler(s ssh.Session) {
	s.Write([]byte(fmt.Sprintf("%s subsytem not implemented\n", s.Subsystem())))
	s.Exit(-1)
}
