package sources

import (
	"log"

	"github.com/awesome-goose/goose/types"
	"github.com/awesome-goose/goose/utils/path"
	"github.com/spf13/viper"
)

type fileEnvSource struct{}

func NewFileEnvSource() *fileEnvSource {
	return &fileEnvSource{}
}

// Load reads the .env file and populates the Env store
func (v *fileEnvSource) Load(env types.Env) {
	directory, err := path.AppRoot()
	if err != nil {
		log.Println("Error reading env directory", err)
		return
	}

	viper.AddConfigPath(directory)
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		log.Println("Error reading env file", err)
		return
	}

	for _, key := range viper.AllKeys() {
		env.Set(key, viper.GetString(key))
	}
}
