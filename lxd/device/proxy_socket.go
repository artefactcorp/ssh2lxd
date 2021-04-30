package device

import (
	"path"
	"strconv"
	"time"

	"ssh2lxd/util"

	"github.com/lxc/lxd/client"
	log "github.com/sirupsen/logrus"
)

type ProxySocket struct {
	Server   *lxd.InstanceServer
	Instance string
	Source   string
	Uid      int
	Gid      int
	Mode     string

	deviceName string
	target     string
}

func (p *ProxySocket) AddProxySocket() (string, error) {
	instance, etag, err := (*p.Server).GetInstance(p.Instance)
	if err != nil {
		log.Errorln(err.Error())
		return "", err
	}

	tmpDir := "/tmp"
	p.deviceName = "proxy-socket-" + strconv.FormatInt(time.Now().UnixNano(), 16) + util.RandomStringLower(5)
	p.target = path.Join(tmpDir, p.deviceName+".sock")

	_, ok := instance.Devices[p.deviceName]
	if ok {
		log.Errorf("device %s already exists for %s", p.deviceName, instance.Name)
		return "", err
	}

	device := map[string]string{}
	device["type"] = "proxy"
	device["connect"] = "unix:" + p.Source
	device["listen"] = "unix:" + p.target
	device["bind"] = "container"
	device["mode"] = p.Mode
	device["uid"] = strconv.Itoa(p.Uid)
	device["gid"] = strconv.Itoa(p.Gid)

	instance.Devices[p.deviceName] = device
	op, err := (*p.Server).UpdateInstance(instance.Name, instance.Writable(), etag)
	if err != nil {
		log.Errorln(err.Error())
		return "", err
	}

	err = op.Wait()
	if err != nil {
		log.Errorln(err.Error())
		return "", err
	}

	return p.target, nil
}

func (p *ProxySocket) RemoveProxySocket() {
	instance, etag, err := (*p.Server).GetInstance(p.Instance)
	if err != nil {
		log.Errorln(err.Error())
		return
	}

	_, ok := instance.Devices[p.deviceName]
	if !ok {
		log.Errorf("device %s does not exist for %s", p.deviceName, instance.Name)
		return
	}
	delete(instance.Devices, p.deviceName)

	op, err := (*p.Server).UpdateInstance(instance.Name, instance.Writable(), etag)
	if err != nil {
		log.Errorln(err.Error())
		return
	}

	err = op.Wait()
	if err != nil {
		log.Errorln(err.Error())
	}
}
