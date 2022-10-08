package main

import (
	"errors"
	"os"
)

func ptIsClient() (bool, error) {
	clientEnv := os.Getenv("TOR_PT_CLIENT_TRANSPORTS")
	serverEnv := os.Getenv("TOR_PT_SERVER_TRANSPORTS")
	if clientEnv != "" && serverEnv != "" {
		return false, errors.New(
			"TOR_PT_[CLIENT,SERVER]_TRANSPORTS both set",
		)
	} else if clientEnv != "" {
		return true, nil
	} else if serverEnv != "" {
		return false, nil
	}
	return false, errors.New("not launched as a managed transport")
}
