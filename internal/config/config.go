package config

import (
	"errors"
	"os"
	"strings"
	"time"

	mapstructure "github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

var ErrFileNotFound = errors.New(" file not found")

type App struct {
	Name string `mapstructure:"name"`
}

type Limits struct {
	LoginAttempts    int           `mapstructure:"login_attempts"`
	PasswordAttempts int           `mapstructure:"password_attempts"`
	IPAttempts       int           `mapstructure:"ip_attempts"`
	Window           time.Duration `mapstructure:"window"`
}

type Server struct {
	Address string `mapstructure:"address"`
	Port    int    `mapstructure:"port"`
	TLS     struct {
		Enabled  bool   `mapstructure:"enabled"`
		CertFile string `mapstructure:"cert_file"`
		KeyFile  string `mapstructure:"key_file"`
	} `mapstructure:"tls"`
}

type Logger struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

type Database struct {
	Workmode   string `mapstructure:"workmode"` // local/external
	Postgresql struct {
		// Параметры подключения могут задаваться либо в dsn, либо, если dsn не задан в следующих полях
		Dsn string `mapstructure:"dsn"`
		// Поля подключения к БД в случае, если dsn не задан
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Name     string `mapstructure:"name"`
		// параметры пула коннектов
		Pool struct {
			// Макс. число открытых соединений от этого процесса (по умолчанию - 20, без ограничений)
			MaxOpenConns int `mapstructure:"max_open_conns"`
			// Макс. число открытых неиспользуемых соединений
			MaxIdleConns int `mapstructure:"max_idle_conns"`
			// Макс. время жизни одного подключения
			ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
			// Макс. время ожидания подключения в пуле
			ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
		} `mapstructure:"pool"`
	} `mapstructure:"postgresql"`
	Redis struct {
		Address  string `mapstructure:"address"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
		Policer  struct {
			ReadTimeout  time.Duration `mapstructure:"read_timeout"`
			WriteTimeout time.Duration `mapstructure:"write_timeout"`
			DialTimeout  time.Duration `mapstructure:"dial_timeout"`
			PoolSize     int           `mapstructure:"pool_size"`
		} `mapstructure:"policer"`
		Subscriber struct {
			ReadTimeout    time.Duration `mapstructure:"read_timeout"`
			PoolSize       int           `mapstructure:"pool_size"`
			SubnetsChannel string        `mapstructure:"subnets_channel"` // ключ для нотификаций об обновлении списков подсетей
		} `mapstructure:"subscriber"`
	} `mapstructure:"redis"`
}

type Config struct {
	App      App      `mapstructure:"app"`
	Limits   Limits   `mapstructure:"limits"`
	Server   Server   `mapstructure:"server"`
	Logger   Logger   `mapstructure:"logger"`
	Database Database `mapstructure:"database"`
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func setDefaults(v *viper.Viper) {
	// Дефолты
	v.SetDefault("database.workmode", "local") // локальный режим - in-memory хранилища
	v.SetDefault("database.postgresql.host", "localhost")
	v.SetDefault("database.postgresql.port", 5432)
	v.SetDefault("database.postgresql.name", "anti_bruteforce")
	v.SetDefault("database.postgresql.pool.max_open_conns", 20)
	v.SetDefault("database.postgresql.pool.max_idle_conns", 10)
	v.SetDefault("database.postgresql.pool.conn_max_lifetime", "1h")
	v.SetDefault("database.postgresql.pool.conn_max_idle_time", "10m")
	v.SetDefault("server.port", 50051)
	v.SetDefault("logger.level", "info")
	v.SetDefault("limits.login_attempts", 10)
	v.SetDefault("limits.password_attempts", 100)
	v.SetDefault("limits.ip_attempts", 1000)
	v.SetDefault("limits.window", "1m")
	v.SetDefault("redis.address", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.policer.dial_timeout", "5s")
	v.SetDefault("redis.policer.read_timeout", "3s")
	v.SetDefault("redis.policer.write_timeout", "3s")
	v.SetDefault("redis.policer.pool_size", 100)
	v.SetDefault("redis.subscriber.read_timeout", "0s")
	v.SetDefault("redis.subscriber.pool_size", 2)
	v.SetDefault("redis.subscriber.subnets_channel", "abf.subnets.updated")

	// Бинды/ для работы без файла конфигурациии без дефолтов, или с нестандартными ключами окружения
	// _ = v.BindEnv("logger.level", "RATELIMITER_LOGGER__LEVEL")
}

func LoadConfig(cfgFilePath string) (*Config, error) {
	v := viper.New()

	// ENV с префиксом ABF (от Anti-Bruteforce), __ вместо . и _ вместо - в ключах
	v.SetEnvPrefix("ABF")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__", "-", "_"))
	v.AutomaticEnv()

	// устанавливаем дефолты и бинды для загрузки из ENV
	setDefaults(v)

	// если конфиг не задан - ищем по стандартным путям
	if cfgFilePath == "" {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/calendar")
	} else {
		if !fileExists(cfgFilePath) {
			return nil, ErrFileNotFound
		}
		v.SetConfigFile(cfgFilePath)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	// как вариант - без контроля наличия файла
	// _ = v.ReadInConfig()

	var cfg Config
	decoderCfg := &mapstructure.DecoderConfig{
		TagName:          "mapstructure",
		Result:           &cfg,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
		),
	}
	dec, err := mapstructure.NewDecoder(decoderCfg)
	if err != nil {
		return nil, err
	}
	if err := dec.Decode(v.AllSettings()); err != nil {
		return nil, err
	}
	return &cfg, nil
}
