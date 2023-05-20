package configs

import (
	"github.com/c0dered273/automation-remote-controller/internal/common/configs"
	"github.com/c0dered273/automation-remote-controller/internal/common/validators"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	configFileType = "yaml"

	// Переменные окружения
	// PORT - порт на котором нужно поднять сервер
	// DOMAIN_NAME - список разрешенных доменных имен для генерации сертификата
	// DATABASE_URI - строка соединения с БД
	// API_SECRET - ключ для подписи jwt
	// CERT_FILE - путь к файлу сертификата
	// PKEY_FILE - путь к файлу приватного ключа
	envVars = []string{
		"PORT",
		"DOMAIN_NAME",
		"DATABASE_URI",
		"API_SECRET",
		"CERT_FILE",
		"PKEY_FILE",
	}
)

// UserAccountConfig настройки приложения
type UserAccountConfig struct {
	Name           string       `mapstructure:"name"`
	Port           string       `mapstructure:"port"`
	DatabaseUri    string       `mapstructure:"database_uri" validate:"required"`
	ApiSecret      string       `mapstructure:"api_secret" validate:"required"`
	CertFile       string       `mapstructure:"cert_file" validate:"required"`
	PKeyFile       string       `mapstructure:"pkey_file" validate:"required"`
	Client         ClientConfig `mapstructure:"client"`
	configs.Logger `mapstructure:"logger"`
}

type ClientConfig struct {
	DomainName []string `mapstructure:"domain_name"`
}

func setDefaults() {
	viper.SetDefault("port", "8080")
	viper.SetDefault("client.domain_name", "c0dered.pro")
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "pretty")
}

func postProcessing() {
	if viper.IsSet("domain_name") {
		viper.RegisterAlias("client.domain_name", "domain_name")
	}
}

func bindPFlags() error {
	pflag.StringP("port", "p", viper.GetString("port"), "Server port")
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		return err
	}
	return nil
}

func bindConfigFile(filename string, configPath []string, logger zerolog.Logger) error {
	viper.SetConfigName(filename)
	viper.SetConfigType(configFileType)
	for _, path := range configPath {
		viper.AddConfigPath(path)
	}
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logger.Error().Msg("config: file not found")
		} else {
			return err
		}
	}
	return nil
}

func bindEnvVars() error {
	for _, env := range envVars {
		err := viper.BindEnv(env)
		if err != nil {
			return err
		}
	}

	return nil
}

func newConfig() (*UserAccountConfig, error) {
	cfg := &UserAccountConfig{}
	err := viper.Unmarshal(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewUserAccountConfig(filename string, configPath []string, logger zerolog.Logger, validator validators.Validator) (*UserAccountConfig, error) {
	setDefaults()

	err := bindConfigFile(filename, configPath, logger)
	if err != nil {
		return nil, err
	}

	err = bindEnvVars()
	if err != nil {
		return nil, err
	}

	err = bindPFlags()
	if err != nil {
		return nil, err
	}

	cfg, nErr := newConfig()
	if nErr != nil {
		return nil, nErr
	}

	postProcessing()

	err = validator.Validate(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
