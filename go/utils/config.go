package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/golang/glog"

	yaml "gopkg.in/yaml.v2"
)

type submitConfigDataCenter struct {
	Name     string `yaml:"name"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Usercert string `yaml:"usercert"`
	Userkey  string `yaml:"userkey"`
	Endpoint string `yaml:"endpoint"`
}

// Configuration load from user config yaml files
type submitConfig struct {
	DC                []submitConfigDataCenter `yaml:"datacenters"`
	ActiveConfig      *submitConfigDataCenter
	CurrentDatacenter string `yaml:"current-datacenter"`
}

// UserHomeDir get user home dierctory on different platforms
func UserHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func getToken(uri string, body []byte) ([]byte, error) {
	req, err := MakeRequest(uri, "POST", bytes.NewBuffer(body), "", nil, nil)
	if err != nil {
		return nil, err
	}
	return GetResponse(req)
}

func token() (string, error) {
	tokenbytes, err := ioutil.ReadFile(filepath.Join(UserHomeDir(), ".paddle", "token_cache"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "previous token not found, fetching a new one...\n")
		// Authenticate to the cloud endpoint
		authJSON := map[string]string{}
		authJSON["username"] = Config.ActiveConfig.Username
		authJSON["password"] = Config.ActiveConfig.Password
		authStr, _ := json.Marshal(authJSON)
		body, err := getToken(Config.ActiveConfig.Endpoint+"/api-token-auth/", authStr)
		if err != nil {
			return "", err
		}
		var respObj interface{}
		if errJSON := json.Unmarshal(body, &respObj); errJSON != nil {
			return "", errJSON
		}
		tokenStr := respObj.(map[string]interface{})["token"].(string)
		err = ioutil.WriteFile(filepath.Join(UserHomeDir(), ".paddle", "token_cache"), []byte(tokenStr), 0600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "write cache token file error: %v", err)
		}
		// Ignore write token error, fetch a new one next time
		return tokenStr, nil
	}
	return string(tokenbytes), nil
}

func parseConfig(configFile string) *submitConfig {
	// ------------------- load paddle config -------------------
	buf, err := ioutil.ReadFile(configFile)
	config := submitConfig{}
	if err == nil {
		yamlErr := yaml.Unmarshal(buf, &config)
		if yamlErr != nil {
			glog.Errorf("load config %s error: %v\n", configFile, yamlErr)
			return nil
		}
		// put active config
		config.ActiveConfig = nil
		for _, item := range config.DC {
			if item.Name == config.CurrentDatacenter {
				config.ActiveConfig = &item
				break
			}
		}
		return &config
	}
	glog.Errorf("config %s error: %v\n", configFile, err)
	return nil
}

// Config is a global varible to store current parsed config object.
var Config = parseConfig(filepath.Join(UserHomeDir(), ".paddle", "config"))