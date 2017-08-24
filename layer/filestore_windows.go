package layer

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// SetOS writes the "os" file to the layer filestore
func (fm *fileMetadataTransaction) SetOS(os string) error {
	if os == "" {
		return nil
	}
	return fm.ws.WriteFile("os", []byte(os), 0644)
}

// GetOS reads the "os" file from the layer filestore
func (fms *fileMetadataStore) GetOS(layer ChainID) (string, error) {
	contentBytes, err := ioutil.ReadFile(fms.getLayerFilename(layer, "os"))
	if err != nil {
		// For backwards compatibility, the os file may not exist. Default to "windows" if missing.
		if os.IsNotExist(err) {
			return "windows", nil
		}
		return "", err
	}
	content := strings.TrimSpace(string(contentBytes))

	if content != "windows" && content != "linux" {
		return "", fmt.Errorf("invalid operating system value: %s", content)
	}

	return content, nil
}
