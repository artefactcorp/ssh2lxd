package ssh2lxd

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"runtime"
	"ssh2lxd/server"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var version = "devel"
var edition = "ce"
var githash = ""

type App struct {
	Name     string
	Version  string
	Edition  string
	GitHash  string
	LongName string
}

var app *App

var (
	idleTimeout = 180 * time.Second

	flagDebug  = false
	flagListen = ":2222"
	flagHelp   = false
	flagSocket = "/var/snap/lxd/common/lxd/unix.socket"
	flagNoauth = false
	flagGroups = "wheel,lxd"

	flagHealthCheck = ""

	flagVersion = false

	allowedGroups []string

	lxdSocket = ""
)

func init() {
	app := &App{}
	app.Name = reflect.TypeOf(App{}).PkgPath()
	app.Edition = edition
	app.Version = version
	app.GitHash = githash
	app.LongName = fmt.Sprintf("%s-%s %s", app.Name, app.Edition, app.Version)
	if app.GitHash != "" {
		app.LongName += fmt.Sprintf(" (%s)", app.GitHash)
	}

	flag.BoolVarP(&flagHelp, "help", "h", flagHelp, "print help")
	flag.BoolVarP(&flagDebug, "debug", "d", flagDebug, "enable debug log")
	flag.BoolVarP(&flagNoauth, "noauth", "", flagDebug, "disable SSH authentication completely")
	flag.BoolVarP(&flagVersion, "version", "v", flagVersion, "print version")
	flag.StringVarP(&flagListen, "listen", "l", flagListen, "listen on :2222 or 127.0.0.1:2222")
	flag.StringVarP(&flagSocket, "socket", "s", flagSocket, "LXD socket or use LXD_SOCKET")
	flag.StringVarP(&flagGroups, "groups", "g", flagGroups, "list of groups members of which allowed to connect")
	flag.StringVarP(&flagHealthCheck, "healthcheck", "", flagHealthCheck, "enable LXD health check every X minutes, e.g. \"5m\"")
	flag.Parse()

	if flagHelp {
		fmt.Printf("%s\n\n", app.LongName)
		flag.PrintDefaults()
		os.Exit(0)
	}

	if flagVersion {
		fmt.Println(app.LongName)
		os.Exit(0)
	}

	lxdSocket = os.Getenv("LXD_SOCKET")
	if lxdSocket == "" {
		lxdSocket = flagSocket
	}

	allowedGroups = strings.Split(flagGroups, ",")

	log.SetOutput(os.Stdout)
	log.SetReportCaller(true)
	if flagDebug {
		log.SetLevel(log.DebugLevel)
		log.SetFormatter(&log.TextFormatter{
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				filename := path.Base(f.File)
				return fmt.Sprintf("> %s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
			},
		})
	} else {
		log.SetLevel(log.ErrorLevel)
	}

	fmt.Printf("Starting %s on %s, LXD socket %s\n", app.LongName, flagListen, lxdSocket)

	config := &server.Config{
		IdleTimeout:   idleTimeout,
		Debug:         flagDebug,
		Listen:        flagListen,
		Socket:        flagSocket,
		Noauth:        flagNoauth,
		Groups:        flagGroups,
		HealthCheck:   flagHealthCheck,
		AllowedGroups: allowedGroups,
		LxdSocket:     lxdSocket,
	}
	server.Run(config)
}

func GetApp() App {
	return *app
}
