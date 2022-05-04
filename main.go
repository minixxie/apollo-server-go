package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/magiconair/properties"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	var configMap map[string]interface{}
	var err error
	if os.Getenv("CONFIG_FORMAT") == "properties" {
		configMap, err = loadConfigProperties()
	} else {
		configMap, err = loadConfigJson()
	}

	if err != nil {
		return
	}

	r := gin.Default()
	r.GET("/services/config", queryServiceConfig(configMap))
	r.GET("/configs/:appId/:cluster/:namespace", queryConfig(configMap))
	r.GET("/configfiles/json/:appId/:cluster/:namespace", queryConfigJSON(configMap))
	r.GET("/notifications/v2", notificationsLongPolling(configMap))
	r.Run(":80")
}

func loadConfigProperties() (map[string]interface{}, error) {
	configMap := map[string]interface{}{}

	// Read config folder
	appIdFolders, err := ioutil.ReadDir("/configs")
	if err != nil {
		log.Printf("Cannot read config folder")
		return nil, err
	}
	for _, appIdFolder := range appIdFolders {
		if !appIdFolder.IsDir() {
			continue
		}
		appId := appIdFolder.Name()
		appConfig := map[string]interface{}{}
		configMap[appId] = appConfig

		// Read appId folder
		clusterFolders, err := ioutil.ReadDir("/configs/" + appId)
		if err != nil {
			log.Printf("Cannot read cluster folder for appId=" + appId)
			return nil, err
		}

		for _, clusterFolder := range clusterFolders {
			if !clusterFolder.IsDir() {
				continue
			}
			clusterName := clusterFolder.Name()
			clusterConfig := map[string]interface{}{}
			appConfig[clusterName] = clusterConfig

			// Read Cluster folder
			namespaceFiles, err := ioutil.ReadDir("/configs/" + appId + "/" + clusterName)
			if err != nil {
				log.Printf("Cannot read cluster folder for appId=" + appId + ", clusterName=" + clusterName)
				return nil, err
			}

			for _, namespaceFile := range namespaceFiles {
				if !strings.HasSuffix(namespaceFile.Name(), ".properties") {
					continue
				}
				namespace := strings.TrimSuffix(namespaceFile.Name(), ".properties")
				p, err := properties.LoadFile("/configs/" + appId + "/" + clusterName + "/" + namespaceFile.Name(), properties.UTF8)
				if err != nil {
					log.Printf("Cannot read namespace file: " + namespaceFile.Name())
					return nil, err
				}

				// convert properties to map[string]interface{}
				configurations := map[string]interface{}{}
				for key, value := range p.Map() {
					configurations[key] = value
				}

				// add namespace config to the config
				clusterConfig[namespace] = map[string]interface{}{
					"releaseKey": namespaceFile.ModTime().Format("20060102150405") + "-7fec91b6d277b5ab",
					"configurations": configurations,
				}
			}
		}
	}

	return configMap, nil
}

func loadConfigJson() (map[string]interface{}, error) {
	file, err := ioutil.ReadFile("/config.json")
	if err != nil {
		log.Printf("Please prepare /config.json")
		return nil, err
	}
	configMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(file), &configMap)
	if err != nil {
		log.Printf("ERR reading /config.json")
		return nil, err
	}
	return configMap, nil
}

func queryConfigValidation(c *gin.Context, configMap map[string]interface{}) (map[string]interface{}, string) {
	appId := c.Param("appId")
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	reqTime := time.Now()

	// {"timestamp":"2020-03-13T15:01:33.845+0800","status":404,"error":"Not Found","message":"Could not load configurations with appId: myAppID, clusterName: myCluster, namespace: myNamespace","path":"/configs/myAppID/myCluster/myNamespace"}
	notFoundResponse := gin.H{
		"timestamp": reqTime.Format("2006-01-02T15:04:05.999Z"), // "2020-03-13T15:01:33.845+0800",
		"status": 404,
		"error": "Not Found",
		"message": fmt.Sprintf("Could not load configurations with appId: %s, clusterName: %s, namespace: %s", appId, cluster, namespace),
		"path": c.Request.URL.Path,
	}

	appObj, ok := configMap[appId]
	if !ok || appObj == nil {
		log.Printf("ERR: appId %s not found", appId)
		c.JSON(404, notFoundResponse)
		return nil, ""
	}

	clusterObj, ok := appObj.(map[string]interface{})[cluster]
	if !ok || clusterObj == nil {
		log.Printf("ERR: cluster %s not found", cluster)
		c.JSON(404, notFoundResponse)
		return nil, ""
	}

	namespaceObj, ok := clusterObj.(map[string]interface{})[namespace]
	if !ok || namespaceObj == nil {
		log.Printf("ERR: namespace %s not found", namespace)
		c.JSON(404, notFoundResponse)
		return nil, ""
	}

	releaseKeyObj, ok := namespaceObj.(map[string]interface{})["releaseKey"]
	if !ok || releaseKeyObj == nil {
		log.Printf("ERR: releaseKey not found")
		c.JSON(404, notFoundResponse)
		return nil, ""
	}
	configurationsObj, ok := namespaceObj.(map[string]interface{})["configurations"]
	if !ok || configurationsObj == nil {
		log.Printf("ERR: configurations not found")
		c.JSON(404, notFoundResponse)
		return nil, ""
	}

	return configurationsObj.(map[string]interface{}), releaseKeyObj.(string)
}

func queryServiceConfig(configMap map[string]interface{}) func(c *gin.Context) {
	return func(c *gin.Context) {
		urlPrefix := os.Getenv("URL_PREFIX")
		if urlPrefix == "" {
			urlPrefix = "http://127.0.0.1:80/"
		}
		c.JSON(200, []gin.H{
			gin.H{
				"appName": "APOLLO-CONFIGSERVICE",
				"instanceId": "fqdn.com:apollo-configservice:80",
				"homepageUrl": urlPrefix,
			},
		})
	}
}

func queryConfig(configMap map[string]interface{}) func(c *gin.Context) {
	return func(c *gin.Context) {
		appId := c.Param("appId")
		cluster := c.Param("cluster")
		namespace := c.Param("namespace")

		configurationsObj, releaseKey := queryConfigValidation(c, configMap)
		if configurationsObj == nil {
			return
		}

		c.JSON(200, gin.H{
			"appId": appId,
			"cluster": cluster,
			"namespaceName": namespace,
			"releaseKey": releaseKey,
			"configurations": configurationsObj,
		})
	}
}

func queryConfigJSON(configMap map[string]interface{}) func(c *gin.Context) {
	return func(c *gin.Context) {
		configurationsObj, _ := queryConfigValidation(c, configMap)
		if configurationsObj == nil {
			return
		}
		c.JSON(200, configurationsObj)
	}
}

func notificationsLongPolling(configMap map[string]interface{}) func(c *gin.Context) {
	return func(c *gin.Context) {
		log.Printf("notificationsLongPolling(): not yet implemented")
		timeout := c.Query("timeout")
		timeoutMS, err := strconv.Atoi(timeout)
		if err != nil {
			timeoutMS = 30000 // default 30 seconds
		}
		time.Sleep(time.Duration(timeoutMS) * time.Millisecond);
		c.JSON(304, gin.H{});
	}
}