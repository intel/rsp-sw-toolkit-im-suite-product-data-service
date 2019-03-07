package configuration

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"

	"encoding/json"
)

//const defaultConfigFilePath = "./configuration.json"

type Configuration struct {
	parsedJson map[string]interface{}
	sectionName string
}

func NewSectionedConfiguration(sectionName string) (*Configuration, error) {
	config := Configuration{}
	config.sectionName = sectionName

	err := loadConfiguration(&config)
	if err != nil {
		return nil, err
	}


	return &config, nil
}

func NewConfiguration() (*Configuration, error) {
	config := Configuration{}

	_, executablePath, _, ok := runtime.Caller(2)
	if ok {
		config.sectionName = path.Base(path.Dir(executablePath))
	}

	err := loadConfiguration(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (config *Configuration) Load(path string) error {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(file, &config.parsedJson)
	return err
}

func (config *Configuration) GetParsedJson() map[string]interface{} {
	return config.parsedJson
}

func (config *Configuration) GetString(path string) (string, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return "", fmt.Errorf("%s not found", path)
		}

		return value, nil
	}

	item := config.getValue(path)

	value, ok := item.(string)
	if !ok {
		return "", fmt.Errorf("Unable to convert value for '%s' to a string: Value='%v'", path, item)
	}

	return value, nil
}

func (config *Configuration) GetInt(path string) (int, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return 0, fmt.Errorf("%s not found", path)
		}

		intValue, err := strconv.Atoi(value)
		if err != nil {
			return 0, fmt.Errorf("Unable to convert value for '%s' to an int: Value='%v'", path, intValue)
		}

		return intValue, nil
	}

	item := config.getValue(path)

	value, ok := item.(float64)
	if !ok {
		return 0, fmt.Errorf("Unable to convert value for '%s' to an int: Value='%v'", path, item)
	}

	return int(value), nil
}

func (config *Configuration) GetFloat(path string) (float64, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return 0, fmt.Errorf("%s not found", path)
		}

		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, fmt.Errorf("Unable to convert value for '%s' to an int: Value='%v'", path, value)
		}

		return floatValue, nil
	}

	item := config.getValue(path)

	value, ok := item.(float64)
	if !ok {
		return 0, fmt.Errorf("Unable to convert value for '%s' to an int: Value='%v'", path, item)
	}

	return value, nil
}

func (config *Configuration) GetBool(path string) (bool, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return false, fmt.Errorf("%s not found", path)
		}

		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return false, fmt.Errorf("Unable to convert value for '%s' to a bool: Value='%v'", path, boolValue)
		}

		return boolValue, nil
	}

	item := config.getValue(path)

	value, ok := item.(bool)
	if !ok {
		return false, fmt.Errorf("Unable to convert value for '%s' to a bool: Value='%v'", path, item)
	}

	return value, nil
}

func (config *Configuration) GetStringSlice(path string) ([]string, error) {
	if !config.pathExistsInConfigFile(path) {
		value, ok := os.LookupEnv(path)
		if !ok {
			return nil, fmt.Errorf("%s not found", path)
		}

		value = strings.Replace(value, "[", "", 1)
		value = strings.Replace(value, "]", "", 1)

		slice := strings.Split(value, ",")
		resultSlice := []string{}
		for _, item := range slice {
			resultSlice = append(resultSlice, strings.Trim(item, " "))
		}

		return resultSlice, nil
	}

	item := config.getValue(path)
	slice := item.([]interface{})

	stringSlice := []string{}
	for _, sliceItem := range slice {
		value, ok := sliceItem.(string)
		if !ok {
			return nil, fmt.Errorf("Unable to convert a value for '%s' to a string: Value='%v'", path, sliceItem)

		}
		stringSlice = append(stringSlice, value)
	}

	return stringSlice, nil
}

func (config *Configuration) getValue(path string) interface{} {
	if config.parsedJson == nil {
		return nil
	}

	if config.sectionName != "" {
		sectionedPath := fmt.Sprintf("%s.%s",config.sectionName,path)
		value := config.getValueFromJson(sectionedPath)
		if value != nil {
			return value
		}
	}

	value := config.getValueFromJson(path)
	return value
}

func (config *Configuration) getValueFromJson(path string) interface{} {
	pathNodes := strings.Split(path, ".")
	if len(pathNodes) == 0 {
		return nil
	}

	var ok bool
	var value interface{}
	jsonNodes := config.parsedJson
	for _, node := range pathNodes {
		if jsonNodes[node] == nil {
			return nil
		}

		item := jsonNodes[node]
		jsonNodes, ok = item.(map[string]interface{})
		if ok {
			continue
		}

		value = item
		break
	}

	return value
}

func loadConfiguration(config *Configuration) error {
	_, filename, _, ok := runtime.Caller(2)
	if !ok {
		log.Print("No caller information")
	}

	absolutePath := path.Join(path.Dir(filename), "configuration.json")

	// By default load local configuration file if it exists
	if _, err := os.Stat(absolutePath); err != nil {
		absolutePath, ok = os.LookupEnv("runtimeConfigPath")
		if !ok {
			absolutePath = "/run/secrets/configuration.json"
		}
		if _, err := os.Stat(absolutePath); err != nil {
			absolutePath = ""
		}
	}

	if absolutePath != "" {
		err := config.Load(absolutePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (config *Configuration) pathExistsInConfigFile(path string) bool {
	if config.sectionName != "" {
		sectionPath := fmt.Sprintf("%s.%s",config.sectionName,path)
		if config.getValue(sectionPath) != nil {
			return true
		}
	}

	if config.getValue(path) != nil {
		return true
	}

	return false
}
