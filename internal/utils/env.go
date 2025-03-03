package utils

import (
	"github.com/labstack/gommon/log"
	"os"
)

func GetEnvFallback(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		log.Debugf("%s=%v\n", key, value)
		return value
	}
	log.Debugf("%s (fallback)=%v\n", key, fallback)
	return fallback
}

func MustGetEnv(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		log.Debugf("%s=%v\n", key, value)
		return value
	}
	panic("mandatory env " + key + " not found")
}
