package docker

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/dotcloud/docker/archive"
	"github.com/dotcloud/docker/utils"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Image struct {
	ID              string    `json:"id"`
	Parent          string    `json:"parent,omitempty"`
	Comment         string    `json:"comment,omitempty"`
	Created         time.Time `json:"created"`
	Container       string    `json:"container,omitempty"`
	ContainerConfig Config    `json:"container_config,omitempty"`
	DockerVersion   string    `json:"docker_version,omitempty"`
	Author          string    `json:"author,omitempty"`
	Config          *Config   `json:"config,omitempty"`
	Architecture    string    `json:"architecture,omitempty"`
	graph           *Graph
	Size            int64
}

func LoadImage(root string) (*Image, error) {
	// Load the json data
	jsonData, err := ioutil.ReadFile(jsonPath(root))
	if err != nil {
		return nil, err
	}
	img := &Image{}

	if err := json.Unmarshal(jsonData, img); err != nil {
		return nil, err
	}
	if err := ValidateID(img.ID); err != nil {
		return nil, err
	}

	if buf, err := ioutil.ReadFile(path.Join(root, "layersize")); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		if size, err := strconv.Atoi(string(buf)); err != nil {
			return nil, err
		} else {
			img.Size = int64(size)
		}
	}

	// Check that the filesystem layer exists
	if stat, err := os.Stat(layerPath(root)); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Couldn't load image %s: no filesystem layer", img.ID)
		}
		return nil, err
	} else if !stat.IsDir() {
		return nil, fmt.Errorf("Couldn't load image %s: %s is not a directory", img.ID, layerPath(root))
	}
	return img, nil
}

func StoreImage(img *Image, jsonData []byte, layerData archive.Archive, root string) error {
	// Check that root doesn't already exist
	if _, err := os.Stat(root); err == nil {
		return fmt.Errorf("Image %s already exists", img.ID)
	} else if !os.IsNotExist(err) {
		return err
	}
	// Store the layer
	layer := layerPath(root)
	if err := os.MkdirAll(layer, 0755); err != nil {
		return err
	}

	// If layerData is not nil, unpack it into the new layer
	if layerData != nil {
		start := time.Now()
		utils.Debugf("Start untar layer")
		if err := archive.Untar(layerData, layer); err != nil {
			return err
		}
		utils.Debugf("Untar time: %vs", time.Now().Sub(start).Seconds())
	}

	// If raw json is provided, then use it
	if jsonData != nil {
		return ioutil.WriteFile(jsonPath(root), jsonData, 0600)
	} else { // Otherwise, unmarshal the image
		jsonData, err := json.Marshal(img)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile(jsonPath(root), jsonData, 0600); err != nil {
			return err
		}
	}

	return StoreSize(img, root)
}

func StoreSize(img *Image, root string) error {
	layer := layerPath(root)

	var totalSize int64 = 0
	filepath.Walk(layer, func(path string, fileInfo os.FileInfo, err error) error {
		totalSize += fileInfo.Size()
		return nil
	})
	img.Size = totalSize

	if err := ioutil.WriteFile(path.Join(root, "layersize"), []byte(strconv.Itoa(int(totalSize))), 0600); err != nil {
		return nil
	}

	return nil
}

func layerPath(root string) string {
	return path.Join(root, "layer")
}

func jsonPath(root string) string {
	return path.Join(root, "json")
}

// TarLayer returns a tar archive of the image's filesystem layer.
func (image *Image) TarLayer(compression archive.Compression) (archive.Archive, error) {
	layerPath, err := image.layer()
	if err != nil {
		return nil, err
	}
	return archive.Tar(layerPath, compression)
}

func (image *Image) ShortID() string {
	return utils.TruncateID(image.ID)
}

func ValidateID(id string) error {
	if id == "" {
		return fmt.Errorf("Image id can't be empty")
	}
	if strings.Contains(id, ":") {
		return fmt.Errorf("Invalid character in image id: ':'")
	}
	return nil
}

func GenerateID() string {
	id := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, id)
	if err != nil {
		panic(err) // This shouldn't happen
	}
	return hex.EncodeToString(id)
}

// Image includes convenience proxy functions to its graph
// These functions will return an error if the image is not registered
// (ie. if image.graph == nil)
func (img *Image) History() ([]*Image, error) {
	var parents []*Image
	if err := img.WalkHistory(
		func(img *Image) error {
			parents = append(parents, img)
			return nil
		},
	); err != nil {
		return nil, err
	}
	return parents, nil
}

func (img *Image) WalkHistory(handler func(*Image) error) (err error) {
	currentImg := img
	for currentImg != nil {
		if handler != nil {
			if err := handler(currentImg); err != nil {
				return err
			}
		}
		currentImg, err = currentImg.GetParent()
		if err != nil {
			return fmt.Errorf("Error while getting parent image: %v", err)
		}
	}
	return nil
}

func (img *Image) GetParent() (*Image, error) {
	if img.Parent == "" {
		return nil, nil
	}
	if img.graph == nil {
		return nil, fmt.Errorf("Can't lookup parent of unregistered image")
	}
	return img.graph.Get(img.Parent)
}

func (img *Image) root() (string, error) {
	if img.graph == nil {
		return "", fmt.Errorf("Can't lookup root of unregistered image")
	}
	return img.graph.imageRoot(img.ID), nil
}

// Return the path of an image's layer
func (img *Image) layer() (string, error) {
	root, err := img.root()
	if err != nil {
		return "", err
	}
	return layerPath(root), nil
}

func (img *Image) getParentsSize(size int64) int64 {
	parentImage, err := img.GetParent()
	if err != nil || parentImage == nil {
		return size
	}
	size += parentImage.Size
	return parentImage.getParentsSize(size)
}

// Build an Image object from raw json data
func NewImgJSON(src []byte) (*Image, error) {
	ret := &Image{}

	utils.Debugf("Json string: {%s}", src)
	// FIXME: Is there a cleaner way to "purify" the input json?
	if err := json.Unmarshal(src, ret); err != nil {
		return nil, err
	}
	return ret, nil
}
