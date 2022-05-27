package config

import (
	"encoding/json"
	"flag"
	"github.com/spf13/viper"
	"log"
	"testing"
	"vsphere-facade/vsphere/protocol"
)

type Config struct {
	Server struct {
		Mode string `mapstructure:"mode"`
		Port int    `mapstructure:"port"`
		Log  struct {
			Path           string `mapstructure:"path"`
			Level          string `mapstructure:"level"`
			MaxSize        int    `mapstructure:"maxSize"`
			MaxBackups     int    `mapstructure:"maxBackups"`
			MaxAge         int    `mapstructure:"maxAge"`
			EnableFullPath bool   `mapstructure:"enableFullPath"`
		}
		Db struct {
			Badger *struct {
				Path string `mapstructure:"path"`
			} `mapstructure:"badger"`
		}
	}

	App struct {
		Token struct {
			Type   string `mapstructure:"type"`
			Secret string `mapstructure:"secret"`
		}
	}

	Vsphere struct {
		Default struct {
			Deployment struct {
				AdapterType *string `mapstructure:"adapterType"`
				DiskMode    *string `mapstructure:"diskMode"`
				Flag        struct {
					EnableLogging *bool `mapstructure:"enableLogging"`
				} `mapstructure:"flag"`
				StoragePolicies map[string]map[string]string `mapstructure:"storagePolicies"`
			} `mapstructure:"deployment"`
			Operation struct {
				ShutdownFirst bool `mapstructure:"shutdownFirst"`
			} `mapstructure:"operation"`
			Callback *protocol.CallbackReq `mapstructure:"callback"`
		}
		Timeout struct {
			Api             int32
			WaitForClone    int32 `mapstructure:"waitForClone"`
			WaitForIP       int32 `mapstructure:"waitForIp"`
			WaitForNet      int32 `mapstructure:"waitForNet"`
			WaitForRelocate int32 `mapstructure:"waitForRelocate"`
		}
		Cache struct {
			Enable          bool `mapstructure:"enable"`
			RefreshDuration int  `mapstructure:"refreshDuration"`
			Ignore          []struct {
				VCID  string   `mapstructure:"vcid"`
				Items []string `mapstructure:"items"`
			}
		}
		RoutineCount struct {
			Operation  int `mapstructure:"operation"`
			Deployment int `mapstructure:"deployment"`
		} `mapstructure:"routineCount"`
	}
}

var G Config

func Setup() {
	testing.Init()
	configDir := flag.String("config", ".", "config file dir")
	flag.Parse()
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")                // name of config file (without extension)
	viper.AddConfigPath("$HOME/.vsphere-facade") // call multiple times to add many search paths
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	viper.AddConfigPath(*configDir)
	Reload()

	b, _ := json.Marshal(G)
	log.Println("读取到的配置: ", string(b))
}

func Reload() {
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("读取配置失败", err)
		return
	}
	err = viper.Unmarshal(&G)
	if err != nil {
		panic(err)
	}
}
