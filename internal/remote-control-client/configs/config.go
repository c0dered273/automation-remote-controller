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

	// Переменные окружения
	// SERVER_ADDR - адрес gRPC сервера к которому необходимо подключиться
	// CA_CERT - путь к коневому сертификату
	// CLIENT_CERT - путь к клиентскому сертификату
	// PLC_URI - строка подключения к контроллеру по протоколу ModBus
	envVars = []string{
		"SERVER_ADDR",
		"CA_CERT",
		"CLIENT_CERT",
		"PLC_URI",
	}
)

// RClientConfig настройки клиентского приложения
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

// Devices перечень устройств, подключенных к контроллеру
type Devices struct {
	// DeviceID Идентификатор устройства, с помощью него осуществляется привязка команды из сообщения к конкретному устройству
	DeviceID string `mapstructure:"device_id"`
	// TagAddress Адрес регистра в контроллере с указанием типа данных
	TagAddress string `mapstructure:"tag_address"`
	// Values значение передаваемое в контроллер
	Values map[string]string `mapstructure:"values"`
}

// Notifications описывает события, генерируемы е контроллером
type Notifications struct {
	// TagAddress адрес регистра, который генерирует события
	TagAddress string `mapstructure:"tag_address"`
	// Text текст события
	Text map[string]string `mapstructure:"text"`
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
	pflag.StringP("server_addr", "a", viper.GetString("server_addr"), "gRPC server address")
	pflag.StringP("plc_uri", "u", viper.GetString("plc_uri"), "PLC connection string")
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
