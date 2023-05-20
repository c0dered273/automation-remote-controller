package configs

// Logger настройки логирования, парсятся из файла настроек
type Logger struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Caller bool   `mapstructure:"caller"`
}
