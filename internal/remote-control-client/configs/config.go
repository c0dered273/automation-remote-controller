package configs

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/c0dered273/automation-remote-controller/pkg/auth"
	"github.com/c0dered273/automation-remote-controller/pkg/configs"
	"github.com/c0dered273/automation-remote-controller/pkg/validators"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	configFileType = "yaml"

	envVars = []string{
		"SERVER_ADDR",
		"CA_CERT",
		"CLIENT_CERT",
		"PLC_URI",
	}
)

type RClientConfig struct {
	Name           string          `mapstructure:"name"`
	ServerAddr     string          `mapstructure:"server_addr"`
	CACert         string          `mapstructure:"ca_cert" validate:"required"`
	ClientCert     string          `mapstructure:"client_cert" validate:"required"`
	TGUsername     string          `validate:"required"`
	CertID         string          `validate:"required"`
	PLCUri         string          `mapstructure:"plc_uri" validate:"required"`
	Devices        []Devices       `mapstructure:"devices" validate:"required"`
	Notifications  []Notifications `mapstructure:"notifications" validate:"required"`
	configs.Logger `mapstructure:"logger"`
}

type Devices struct {
	DeviceID   string            `mapstructure:"device_id"`
	TagAddress string            `mapstructure:"tag_address"`
	Values     map[string]string `mapstructure:"values"`
}

type Notifications struct {
	TagAddress string            `mapstructure:"tag_address"`
	Text       map[string]string `mapstructure:"text"`
}

func setDefaults() {
	viper.SetDefault("server_addr", "8080")
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "pretty")
}

func postProcessing(config *RClientConfig) error {
	clientPEM, err := os.ReadFile(config.ClientCert)
	if err != nil {
		return fmt.Errorf("failed to read client certificate: %w", err)
	}
	clientCert, err := auth.ParseCert(clientPEM)
	if err != nil {
		return fmt.Errorf("failed to read client certificate: %w", err)
	}
	var tgName string
	var certID string
	for _, n := range clientCert.Subject.Names {
		if reflect.DeepEqual(n.Type, auth.OwnerOID) {
			tgName = n.Value.(string)
			continue
		}
		if reflect.DeepEqual(n.Type, auth.X500UniqueIdentifier) {
			certID = n.Value.(string)
			continue
		}
	}
	if len(tgName) == 0 || len(certID) == 0 {
		return errors.New("client config: failed to parse credentials")
	}

	config.TGUsername = tgName
	config.CertID = certID
	return nil
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

func newConfig() (*RClientConfig, error) {
	cfg := &RClientConfig{}
	err := viper.Unmarshal(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewRClientConfig(filename string, configPath []string, logger zerolog.Logger, validator validators.Validator) (*RClientConfig, error) {
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

	err = postProcessing(cfg)
	if err != nil {
		return nil, err
	}

	err = validator.Validate(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
