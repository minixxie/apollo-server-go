package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func queryConfigValidation(c *gin.Context, configMap *Config) (map[string]interface{}, string) {
	appId := c.Param("appId")
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")

	reqTime := time.Now()

	// {"timestamp":"2020-03-13T15:01:33.845+0800","status":404,"error":"Not Found","message":"Could not load configurations with appId: myAppID, clusterName: myCluster, namespace: myNamespace","path":"/configs/myAppID/myCluster/myNamespace"}
	notFoundResponse := gin.H{
		"timestamp": reqTime.Format("2006-01-02T15:04:05.999Z"), // "2020-03-13T15:01:33.845+0800",
		"status":    404,
		"error":     "Not Found",
		"message":   fmt.Sprintf("Could not load configurations with appId: %s, clusterName: %s, namespace: %s", appId, cluster, namespace),
		"path":      c.Request.URL.Path,
	}

	appObj, ok := (*configMap)[appId]
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

func serviceConfigHandler(configLoader ConfigLoader) func(c *gin.Context) {
	return func(c *gin.Context) {
		urlPrefix := os.Getenv("URL_PREFIX")
		if urlPrefix == "" {
			urlPrefix = "http://127.0.0.1:80/"
		}
		c.JSON(200, []gin.H{
			gin.H{
				"appName":     "APOLLO-CONFIGSERVICE",
				"instanceId":  "fqdn.com:apollo-configservice:80",
				"homepageUrl": urlPrefix,
			},
		})
	}
}

func configHandler(configLoader ConfigLoader) func(c *gin.Context) {
	return func(c *gin.Context) {
		appId := c.Param("appId")
		cluster := c.Param("cluster")
		namespace := c.Param("namespace")

		configMap, err := configLoader.LoadConfig()
		fmt.Printf("%v", configMap)
		if err != nil {
			return
		}

		configurationsObj, releaseKey := queryConfigValidation(c, configMap)
		if configurationsObj == nil {
			return
		}

		c.JSON(200, gin.H{
			"appId":          appId,
			"cluster":        cluster,
			"namespaceName":  namespace,
			"releaseKey":     releaseKey,
			"configurations": configurationsObj,
		})
	}
}

func configJSONHandler(configLoader ConfigLoader) func(c *gin.Context) {
	return func(c *gin.Context) {
		configMap, err := configLoader.LoadConfig()
		if err != nil {
			return
		}
		configurationsObj, _ := queryConfigValidation(c, configMap)
		if configurationsObj == nil {
			return
		}
		c.JSON(200, configurationsObj)
	}
}

func notificationsLongPollingHandler(configLoader ConfigLoader) func(c *gin.Context) {
	return func(c *gin.Context) {
		log.Printf("notificationsLongPollingHandler(): not yet implemented")
		timeout := c.Query("timeout")
		timeoutMS, err := strconv.Atoi(timeout)
		if err != nil {
			timeoutMS = 30000 // default 30 seconds
		}
		time.Sleep(time.Duration(timeoutMS) * time.Millisecond)
		c.JSON(304, gin.H{})
	}
}
