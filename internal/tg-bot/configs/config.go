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

	envVars = []string{
		"PORT",
		"BOT_TOKEN",
		"CA_CERT",
		"SERVER_CERT",
		"SERVER_PKey",
		"DATABASE_URI",
	}
)

type TGBotCfg struct {
	Name           string `mapstructure:"name"`
	Port           string `mapstructure:"port"`
	BotToken       string `mapstructure:"bot_token" validate:"required"`
	CACert         string `mapstructure:"ca_cert" validate:"required"`
	ServerCert     string `mapstructure:"server_cert" validate:"required"`
	ServerPkey     string `mapstructure:"server_pkey" validate:"required"`
	DatabaseUri    string `mapstructure:"database_uri" validate:"required"`
	configs.Logger `mapstructure:"logger"`
}

func setDefaults() {
	viper.SetDefault("port", "8080")
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "pretty")
}

func postProcessing() {
}

func bindPFlags() error {
	pflag.StringP("port", "p", viper.GetString("port"), "Server port")
	pflag.StringP("bot_token", "t", viper.GetString("bot_token"), "Token from @Botfather")
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

func newConfig() (*TGBotCfg, error) {
	cfg := &TGBotCfg{}
	err := viper.Unmarshal(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewTGBotConfig(filename string, configPath []string, logger zerolog.Logger, validator validators.Validator) (*TGBotCfg, error) {
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
