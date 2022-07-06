package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

const (
	CONFIG_FORMAT_JSON       = "json"
	CONFIG_FORMAT_PROPERTIES = "properties"
)

var port string
var configFormat string
var configFilePath string

func init() {
	flag.StringVar(&configFormat, "configFormat", "json", "config format: json (default) | properties")
	flag.StringVar(&configFilePath, "configFilePath", "/config.json", "config file path. default /config.json")

	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	} else {
		port = "80"
	}
}

func main() {
	flag.Parse()

	var configLoader ConfigLoader
	switch configFormat {
	case CONFIG_FORMAT_PROPERTIES:
		configLoader = PropertiesConfigLoader{}
	case CONFIG_FORMAT_JSON:
		fallthrough
	default:
		configLoader = JSONConfigLoader{}
	}

	r := gin.Default()
	r.GET("/services/config", serviceConfigHandler(configLoader))
	r.GET("/configs/:appId/:cluster/:namespace", configHandler(configLoader))
	r.GET("/configfiles/json/:appId/:cluster/:namespace", configJSONHandler(configLoader))
	r.GET("/notifications/v2", notificationsLongPollingHandler(configLoader))
	r.Run(fmt.Sprintf(":%s", port))
}
