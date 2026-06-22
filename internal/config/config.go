package config

import (
    "github.com/joho/godotenv"
    "github.com/spf13/viper"
    "log"
)

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    GeoIP    GeoIPConfig
    App      AppConfig
}

type ServerConfig struct {
    Port         string
    ReadTimeout  int
    WriteTimeout int
}

type DatabaseConfig struct {
    URL         string
    MaxConns    int
    MinConns    int
}

type RedisConfig struct {
    Addr     string
    Password string
    DB       int
    CacheTTL int // seconds
}

type GeoIPConfig struct {
    DBPath string
}

type AppConfig struct {
    BaseURL       string
    ShortCodeLen  int
    DefaultExpiry int // days, 0 = no expiry
}

func Load() *Config {
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found, reading from environment")
    }

    viper.AutomaticEnv()

    // Server defaults
    viper.SetDefault("SERVER_PORT", "8080")
    viper.SetDefault("SERVER_READ_TIMEOUT", 10)
    viper.SetDefault("SERVER_WRITE_TIMEOUT", 30)

    // Redis defaults
    viper.SetDefault("REDIS_ADDR", "localhost:6379")
    viper.SetDefault("REDIS_DB", 0)
    viper.SetDefault("REDIS_CACHE_TTL", 3600)

    // App defaults
    viper.SetDefault("APP_SHORT_CODE_LEN", 7)
    viper.SetDefault("APP_DEFAULT_EXPIRY", 0)

    return &Config{
        Server: ServerConfig{
            Port:         viper.GetString("SERVER_PORT"),
            ReadTimeout:  viper.GetInt("SERVER_READ_TIMEOUT"),
            WriteTimeout: viper.GetInt("SERVER_WRITE_TIMEOUT"),
        },
        Database: DatabaseConfig{
            URL:      viper.GetString("DATABASE_URL"),
            MaxConns: viper.GetInt("DB_MAX_CONNS"),
            MinConns: viper.GetInt("DB_MIN_CONNS"),
        },
        Redis: RedisConfig{
            Addr:     viper.GetString("REDIS_ADDR"),
            Password: viper.GetString("REDIS_PASSWORD"),
            DB:       viper.GetInt("REDIS_DB"),
            CacheTTL: viper.GetInt("REDIS_CACHE_TTL"),
        },
        GeoIP: GeoIPConfig{
            DBPath: viper.GetString("GEOIP_DB_PATH"),
        },
        App: AppConfig{
            BaseURL:       viper.GetString("APP_BASE_URL"),
            ShortCodeLen:  viper.GetInt("APP_SHORT_CODE_LEN"),
            DefaultExpiry: viper.GetInt("APP_DEFAULT_EXPIRY"),
        },
    }
}