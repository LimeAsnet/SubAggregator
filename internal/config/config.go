package config

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

const configNameDefault = "local"

type HttpServer struct {
	Host string `mapstructure:"srvHost"`
}

type Database struct {
	Name     string `mapstructure:"dbName"`
	Host     string `mapstructure:"dbHost"`
	Port     string `mapstructure:"dbPort"`
	SSLmode  string `mapstructure:"dbSSLmode"`
	User     string `mapstructure:"dbUser"`
	Password string `mapstructure:"-"` // только из переменных окружения
}

type Config struct {
	Env        string `mapstructure:"env"`
	HttpServer `mapstructure:"httpServer"`
	Database   `mapstructure:"database"`
}

func (d Database) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLmode,
	)
}

func InitConfig() Config {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}
	return cfg
}

func LoadConfig() (Config, error) {
	return LoadConfigFromDir("./internal/config", os.Getenv("CONFIG_NAME"))
}

func LoadConfigFromDir(configDir, configName string) (Config, error) {
	_ = gotenv.Load(".env")

	if configName == "" {
		configName = configNameDefault
	}

	viperCfg := viper.New()
	viperCfg.SetConfigName(configName)
	viperCfg.SetConfigType("yaml")
	viperCfg.AddConfigPath(configDir)

	if err := viperCfg.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config %q from %s: %w", configName, configDir, err)
	}

	var cfg Config
	if err := viperCfg.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.Database.Password = os.Getenv("DB_PASSWORD")
	if cfg.Database.Password == "" {
		cfg.Database.Password = os.Getenv("POSTGRES_PASSWORD")
	}
	if cfg.Database.Password == "" {
		return Config{}, fmt.Errorf("set DB_PASSWORD or POSTGRES_PASSWORD in .env (see .env.example)")
	}

	return cfg, nil
}
