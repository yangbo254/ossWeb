package ossweb

import (
	"encoding/json"
	"errors"
	"log"
	"os"
)

type config struct {
	kvMap map[string]interface{}
}

var gc *config

func RunConfig() {
	gc = &config{}
	content, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}
	if err = json.Unmarshal(content, &gc.kvMap); err != nil {
		gc.kvMap = make(map[string]interface{})
		log.Fatal(err)
	}
}

func GetConfigValue(key string) (interface{}, error) {
	if v, found := gc.kvMap[key]; found {
		return v, nil
	}
	return nil, errors.New("not found key in config")
}
