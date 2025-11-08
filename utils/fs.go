package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func ResolveFileInDefaultConfigDir(
	name string,
	createFile func(fullPath string) error,
) (fullPath string, err error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	fullPath = filepath.Join(configDir, name)

	fmt.Println(fullPath)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		if err := createFile(fullPath); err != nil {
			return fullPath, err
		}
	}

	return fullPath, nil
}
