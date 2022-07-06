package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"

	"github.com/magiconair/properties"
)

type Config map[string]interface{}

type ConfigLoader interface {
	LoadConfig() (*Config, error)
}

type JSONConfigLoader struct{}

func (jcl JSONConfigLoader) LoadConfig() (*Config, error) {
	return loadConfigJson()
}

type PropertiesConfigLoader struct{}

func (pcl PropertiesConfigLoader) LoadConfig() (*Config, error) {
	return loadConfigProperties()
}

func loadConfigProperties() (*Config, error) {
	configMap := Config{}

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
				p, err := properties.LoadFile("/configs/"+appId+"/"+clusterName+"/"+namespaceFile.Name(), properties.UTF8)
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
					"releaseKey":     namespaceFile.ModTime().Format("20060102150405") + "-7fec91b6d277b5ab",
					"configurations": configurations,
				}
			}
		}
	}

	return &configMap, nil
}

func loadConfigJson() (*Config, error) {
	file, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		log.Printf("Please prepare %s", configFilePath)
		return nil, err
	}
	configMap := Config{}
	err = json.Unmarshal([]byte(file), &configMap)
	if err != nil {
		log.Printf("ERR reading %s", configFilePath)
		return nil, err
	}
	return &configMap, nil
}
