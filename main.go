package main

import (
	"main/ossweb"

	log "github.com/sirupsen/logrus"
)

func main() {
	ossweb.RunConfig()
	web, err := ossweb.NewWebEngine()
	if err != nil {
		log.Error(err)
		return
	}
	_ = web.Run()
}
