// Package imstor enables you to create copies (or thumbnails) of your images and stores
// them along with the original image on your filesystem. The image and its
// copies are are stored in a file structure based on the (zero-prefixed, decimal)
// CRC 64 checksum of the original image. The last 2 characters of the checksum
// are used as the lvl 1 directory name.
//
// Example folder name and contents, given the checksum 08446744073709551615 and
// sizes named "small" and "large":
//
//  /configured/root/path/15/08446744073709551615/original.jpeg
//  /configured/root/path/15/08446744073709551615/small.jpeg
//  /configured/root/path/15/08446744073709551615/large.jpeg
package imstor

import (
	"errors"
	"fmt"
	"hash/crc64"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
	"crypto/rand"

	"github.com/vincent-petithory/dataurl"
)

var crcTable = crc64.MakeTable(crc64.ISO)

const (
	originalImageName = "original"
)

type storage struct {
	conf    *Config
	resizer Resizer
}

// Storage is the engine that can be used to store images and retrieve their paths
type Storage interface {
	Store(mediaType string, data []byte) (string, error)
	StoreDataURL(string) (string, error)
	Checksum([]byte) string
	ChecksumDataURL(string) (string, error)
	PathFor(checksum string) (string, error)
	PathForSize(checksum, size string) (string, error)
}

// New creates a storage engine using the default Resizer
func New(conf *Config) Storage {
	return storage{
		conf:    conf,
		resizer: DefaultResizer,
	}
}

// NewWithCustomResizer creates a storage engine using a custom resizer
func NewWithCustomResizer(conf *Config, resizer Resizer) Storage {
	return storage{
		conf:    conf,
		resizer: resizer,
	}
}

func getStructuredFolderPath(checksum string) string {
	lvl1Dir := checksum[len(checksum)-3:]
	return path.Join(lvl1Dir, checksum)
}

func (s storage) ChecksumDataURL(str string) (string, error) {
	dataURL, err := dataurl.DecodeString(str)
	if err != nil {
		return "", err
	}
	return s.Checksum(dataURL.Data), nil
}

func (s storage) Checksum(data []byte) string {
	return NewUUIDv4()
}

func (s storage) PathFor(sum string) (string, error) {
	return s.PathForSize(sum, originalImageName)
}

func (s storage) PathForSize(sum, size string) (string, error) {
	dir := getStructuredFolderPath(sum)
	absDirPath := filepath.Join(s.conf.RootPath, filepath.FromSlash(dir))
	files, err := ioutil.ReadDir(absDirPath)
	if err != nil {
		return "", err
	}
	for _, file := range files {
		if !file.IsDir() && hasNameWithoutExtension(file.Name(), size) {
			return filepath.Join(dir, file.Name()), nil
		}
	}
	return "", errors.New("File not found!")
}

func hasNameWithoutExtension(fileName, name string) bool {
	extension := path.Ext(fileName)
	nameWithoutExtension := strings.TrimSuffix(fileName, extension)
	return nameWithoutExtension == name
}

func NewUUIDv4() string {
	u := [16]byte{}
	_, err := rand.Read(u[:16])
	if err != nil {
		panic(err)
	}

	u[8] = (u[8] | 0x80) & 0xBf
	u[6] = (u[6] | 0x40) & 0x4f

	return fmt.Sprintf("%x-%x-%x-%x-%x", u[:4], u[4:6], u[6:8], u[8:10], u[10:])
}
