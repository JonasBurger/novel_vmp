package config

import (
	"log"

	"github.com/spf13/viper"
)

func LoadViperConfig() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Using config file:", viper.ConfigFileUsed())
}
