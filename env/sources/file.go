package sources

import (
	"os"

	"github.com/awesome-goose/goose/types"
	"github.com/awesome-goose/goose/utils/path"
	"github.com/spf13/viper"
)

type fileEnvSource struct{}

func NewFileEnvSource() *fileEnvSource {
	return &fileEnvSource{}
}

// Load reads the .env file and populates the Env store
// Silently ignores missing .env files
func (v *fileEnvSource) Load(env types.Env) {
	directory, err := path.AppRoot()
	if err != nil {
		return
	}

	envPath := directory + "/.env"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return
	}

	viper.AddConfigPath(directory)
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return // Silently ignore read errors
	}

	for _, key := range viper.AllKeys() {
		env.Set(key, viper.GetString(key))
	}
}
