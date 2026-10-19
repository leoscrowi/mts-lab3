package config

import (
	"os"
	"time"
)

type Config struct {
	RestBaseURL string
	GraphQLURL  string
	Timeout     time.Duration
}

func LoadConfig() Config {
	return Config{
		RestBaseURL: env("REST_BASE_URL", "https://playground.nileslabs.com/api/v1"),
		GraphQLURL:  env("GRAPHQL_URL", "https://graphql.postman-echo.com/graphql"),
		Timeout:     duration("HTTP_TIMEOUT", 20*time.Second),
	}
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func duration(key string, def time.Duration) time.Duration {
	if d, err := time.ParseDuration(env(key, "")); err == nil {
		return d
	}
	return def
}
