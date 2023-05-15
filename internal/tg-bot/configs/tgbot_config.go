package configs

import (
	"github.com/c0dered273/automation-remote-controller/internal/common/validators"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	configFileType = "yaml"

	envVars = []string{
		"BOT_TOKEN",
	}
)

type TGBotCfg struct {
	Name     string `mapstructure:"name"`
	BotToken string `mapstructure:"bot_token" validate:"required"`
	Logger   `mapstructure:"logger"`
}

type Logger struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Caller bool   `mapstructure:"caller"`
}

func setDefaults() {
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "pretty")
}

func bindPFlags() error {
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

func NewTGBotConfiguration(filename string, configPath []string, logger zerolog.Logger, validator *validator.Validate) (*TGBotCfg, error) {
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

	err = validators.ValidateStructWithLogger(cfg, logger, validator)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
