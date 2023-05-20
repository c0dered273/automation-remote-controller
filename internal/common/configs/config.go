package configs

type Logger struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Caller bool   `mapstructure:"caller"`
}
