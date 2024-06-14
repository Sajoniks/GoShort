package config

import "os"

func GetEnvironment() string {
	env := os.Getenv("GOSHRT_ENV")
	return env
}
