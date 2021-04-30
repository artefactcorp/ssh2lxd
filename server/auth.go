package server

import (
	"ssh2lxd/util/ssh"

	log "github.com/sirupsen/logrus"
)

func keyAuthHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	var user string

	user, _, _ = parseUser(ctx.User())

	osUser, err := getOsUser(user)
	if err != nil {
		return false
	}

	if len(config.AllowedGroups) > 0 {
		userGroups, err := getUserGroups(osUser)
		if err != nil {
			return false
		}
		if !isGroupMatch(config.AllowedGroups, userGroups) {
			log.Debugf("no group match for %s in %v", user, userGroups)
			return false
		}
	}

	keys, _ := getUserAuthKeys(osUser)
	for _, k := range keys {
		pk, _, _, _, err := ssh.ParseAuthorizedKey(k)
		if err != nil {
			log.Debugln(err.Error())
			continue
		}
		if ssh.KeysEqual(pk, key) {
			return true
		}
	}

	log.Debugf("key auth failed for %s", user)
	return false
}
