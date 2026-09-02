package systemtests

import (
	"errors"
	"strings"
)

func authorizationBootstrapArgs(userID string) ([]string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" || strings.HasPrefix(userID, "-") {
		return nil, errors.New("bootstrap user id is required")
	}
	return []string{
		"compose", "-f", "../environments/local/docker-compose.yml", "exec", "-T",
		"authorization-service", "/app/bootstrap-admin", "--config", "/app/config/config.yaml",
		"--env", "compose", "--user-id", userID,
	}, nil
}
