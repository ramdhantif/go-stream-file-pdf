package config

import (
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Reader interface {
	Get(key string) string
}

type viperConfigReader struct {
	viper *viper.Viper
}

var Data *viperConfigReader

func (v viperConfigReader) Get(key string) string {
	return v.viper.GetString(key)
}

func (v viperConfigReader) GetInt(key string) int {
	return v.viper.GetInt(key)
}

func Load() {
	v := viper.New()
	v.AddConfigPath(".")
	v.SetConfigName("conf")
	v.SetConfigType("yaml")
	v.AutomaticEnv()

	err := v.ReadInConfig()
	if err != nil {
		return
	}
	Data = &viperConfigReader{
		viper: v,
	}
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		log.Println("config file changed", e.Name)
	})
	return
}
