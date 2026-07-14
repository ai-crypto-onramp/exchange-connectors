package otc

import (
	"fmt"
	"os"
	"path/filepath"
)

func fetchSFTP(basePath, orderID string) ([]byte, error) {
	if basePath == "" {
		return nil, fmt.Errorf("otc: sftp path not configured")
	}
	path := filepath.Join(basePath, orderID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("otc: sftp read: %w", err)
	}
	return data, nil
}