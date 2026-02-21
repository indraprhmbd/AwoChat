package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Session  SessionConfig
	Limits   LimitsConfig
}

type ServerConfig struct {
	Host string
	Port string
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type SessionConfig struct {
	CookieName     string
	CookieSecure   bool
	CookieHTTPOnly bool
	CookieSameSite string
	Expiration     time.Duration
}

type LimitsConfig struct {
	MaxRoomMembers     int
	MaxMessageSize     int
	MaxSendBufferSize  int
	SignupRateLimit    int
	LoginRateLimit     int
	MessageRateLimit   int
	TypingThrottleSec  int
	MaxRoomsPerUser    int
	MaxSessionsPerUser int
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "awochat"),
			Password:        getEnv("DB_PASSWORD", "awochat"),
			DBName:          getEnv("DB_NAME", "awochat"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: 5 * time.Minute,
		},
		Session: SessionConfig{
			CookieName:     getEnv("SESSION_COOKIE_NAME", "awochat_session"),
			CookieSecure:   getEnvBool("SESSION_COOKIE_SECURE", false),
			CookieHTTPOnly: getEnvBool("SESSION_COOKIE_HTTP_ONLY", true),
			CookieSameSite: getEnv("SESSION_COOKIE_SAME_SITE", "Strict"),
			Expiration:     24 * time.Hour,
		},
		Limits: LimitsConfig{
			MaxRoomMembers:     getEnvInt("MAX_ROOM_MEMBERS", 100),
			MaxMessageSize:     getEnvInt("MAX_MESSAGE_SIZE", 2048),
			MaxSendBufferSize:  getEnvInt("MAX_SEND_BUFFER_SIZE", 32),
			SignupRateLimit:    getEnvInt("SIGNUP_RATE_LIMIT", 5),
			LoginRateLimit:     getEnvInt("LOGIN_RATE_LIMIT", 10),
			MessageRateLimit:   getEnvInt("MESSAGE_RATE_LIMIT", 10),
			TypingThrottleSec:  getEnvInt("TYPING_THROTTLE_SEC", 1),
			MaxRoomsPerUser:    getEnvInt("MAX_ROOMS_PER_USER", 50),
			MaxSessionsPerUser: getEnvInt("MAX_SESSIONS_PER_USER", 10),
		},
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}
