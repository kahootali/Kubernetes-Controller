package config

import (
	"io/ioutil"
	"log"

	yaml "gopkg.in/yaml.v2"
)

type Configuration struct {
	ApiVersion string
	Kind       string
	Metadata   MetaData
	Spec       Specifications
}
type Specifications struct {
	DeploymentName string
	Replicas       int
}
type MetaData struct {
	Name string
}

//ReadConfig function that reads the yaml file
func ReadConfig(filePath string) Configuration {
	var config Configuration
	// Read YML
	source, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Panic(err)
	}

	// Unmarshall
	err = yaml.Unmarshal(source, &config)
	if err != nil {
		log.Panic(err)
	}

	return config
}

//WriteConfig function that can write to the yaml file
func WriteConfig(config Configuration, path string) error {
	b, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, b, 0644)
	if err != nil {
		return err
	}

	return nil
}
