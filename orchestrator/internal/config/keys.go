package config

import (
	"log"

	"os"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var Keymap map[string]string

func InitKeys() {
	if Keymap != nil {
		log.Panicf("Keys already initialized")
	}
	keyfile := viper.GetString("keyfile")
	if keyfile == "" {
		log.Panicf("Keyfile not set")
	}
	// load contents from keyfile into keymap
	fileContents, err := os.ReadFile(keyfile)
	if err != nil {
		log.Panicf("Failed to read keyfile: %v", err)
	}
	// load yaml contents from keyfile into keymap (without viper)
	Keymap = make(map[string]string)
	err = yaml.Unmarshal(fileContents, &Keymap)
	if err != nil {
		log.Panicf("Failed to unmarshal keyfile contents: %v", err)
	}
}
