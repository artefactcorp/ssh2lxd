package server

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	hostuser "ssh2lxd/util/user"

	log "github.com/sirupsen/logrus"
)

func getOsUser(user string) (*hostuser.User, error) {
	u, err := hostuser.Lookup(user)
	if err != nil {
		log.Errorln(err.Error())
		return nil, err
	}
	return u, nil
}

func getUserAuthKeys(u *hostuser.User) ([][]byte, error) {
	var keys [][]byte

	f, err := os.Open(filepath.Clean(u.HomeDir + "/.ssh/authorized_keys"))
	if err != nil {
		log.Errorln(err.Error())
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		keys = append(keys, s.Bytes())
	}
	return keys, nil
}

func getUserGroups(u *hostuser.User) ([]string, error) {
	groups, err := u.GroupIds()
	if err != nil {
		log.Errorln(err.Error())
		return nil, err
	}
	return groups, nil
}

func parseUser(user string) (string, string, string) {
	var instance string
	var instanceUser = "root"

	if strings.Contains(user, "+") {
		uu := strings.Split(user, "+")
		user = uu[0]
		instance = uu[1]
		if len(uu) > 2 {
			instanceUser = uu[2]
		}
	} else {
		instance = user
		user = "root"
	}

	return user, instance, instanceUser
}

func getGroupIds(groups []string) []string {
	var ids []string
	for _, g := range groups {
		group, err := hostuser.LookupGroup(g)
		if err != nil {
			log.Errorln(err.Error())
			continue
		}
		ids = append(ids, group.Gid)
	}
	return ids
}

func isGroupMatch(a []string, b []string) bool {
	for _, i := range a {
		for _, j := range b {
			if i == j {
				return true
			}
		}
	}
	return false
}
