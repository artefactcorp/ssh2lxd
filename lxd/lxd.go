package lxd

import (
	"ssh2lxd/util/structs"

	"github.com/lxc/lxd/client"
)

func Connect(socket string) (lxd.InstanceServer, error) {
	return lxd.ConnectLXDUnix(socket, nil)
}

func GetConnectionInfo(c lxd.InstanceServer) map[string]interface{} {
	info, _ := c.GetConnectionInfo()
	return structs.Map(info)
}
