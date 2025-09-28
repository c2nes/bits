package main

import (
	"os"
	"path/filepath"
)

func HistoryFile() (string, error) {
	// See https://specifications.freedesktop.org/basedir-spec/latest/
	xdgStateHome, _ := os.LookupEnv("XDG_STATE_HOME")
	if xdgStateHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		xdgStateHome = filepath.Join(homeDir, ".local/state")
	}
	historyFile := filepath.Join(xdgStateHome, "bits/history")
	historyDir := filepath.Dir(historyFile)
	if err := os.MkdirAll(historyDir, 0777); err != nil {
		return "", err
	}
	return historyFile, nil
}
