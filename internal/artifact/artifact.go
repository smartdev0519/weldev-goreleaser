// Package artifact provides the core artifact storage for goreleaser
package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"sync"

	"github.com/apex/log"
)

// Type defines the type of an artifact
type Type int

const (
	// UploadableArchive a tar.gz/zip archive to be uploaded
	UploadableArchive Type = iota
	// UploadableBinary is a binary file to be uploaded
	UploadableBinary
	// Binary is a binary (output of a gobuild)
	Binary
	// LinuxPackage is a linux package generated by nfpm or snapcraft
	LinuxPackage
	// DockerImage is a docker image
	DockerImage
	// Checksum is a checksums file
	Checksum
	// Signature is a signature file
	Signature
)

func (t Type) String() string {
	switch t {
	case UploadableArchive:
		return "Archive"
	case UploadableBinary:
		return "Binary"
	case Binary:
		return "Binary"
	case LinuxPackage:
		return "LinuxPackage"
	case DockerImage:
		return "DockerImage"
	case Checksum:
		return "Checksum"
	case Signature:
		return "Signature"
	}
	return "unknown"
}

// Artifact represents an artifact and its relevant info
type Artifact struct {
	Name   string
	Path   string
	Goos   string
	Goarch string
	Goarm  string
	Type   Type
	Extra  map[string]string
}

// Checksum calculates the SHA256 checksum of the artifact.
func (a Artifact) Checksum() (string, error) {
	log.Debugf("calculating sha256sum for %s", a.Path)
	file, err := os.Open(a.Path)
	if err != nil {
		return "", err
	}
	defer file.Close() // nolint: errcheck
	var hash = sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Artifacts is a list of artifacts
type Artifacts struct {
	items []Artifact
	lock  *sync.Mutex
}

// New return a new list of artifacts
func New() Artifacts {
	return Artifacts{
		items: []Artifact{},
		lock:  &sync.Mutex{},
	}
}

// List return the actual list of artifacts
func (artifacts Artifacts) List() []Artifact {
	return artifacts.items
}

// GroupByPlatform groups the artifacts by their platform
func (artifacts Artifacts) GroupByPlatform() map[string][]Artifact {
	var result = map[string][]Artifact{}
	for _, a := range artifacts.items {
		plat := a.Goos + a.Goarch + a.Goarm
		result[plat] = append(result[plat], a)
	}
	return result
}

// Add safely adds a new artifact to an artifact list
func (artifacts *Artifacts) Add(a Artifact) {
	artifacts.lock.Lock()
	defer artifacts.lock.Unlock()
	log.WithFields(log.Fields{
		"name": a.Name,
		"path": a.Path,
		"type": a.Type,
	}).Info("added new artifact")
	artifacts.items = append(artifacts.items, a)
}

// Filter defines an artifact filter which can be used within the Filter
// function
type Filter func(a Artifact) bool

// ByGoos is a predefined filter that filters by the given goos
func ByGoos(s string) Filter {
	return func(a Artifact) bool {
		return a.Goos == s
	}
}

// ByGoarch is a predefined filter that filters by the given goarch
func ByGoarch(s string) Filter {
	return func(a Artifact) bool {
		return a.Goarch == s
	}
}

// ByGoarm is a predefined filter that filters by the given goarm
func ByGoarm(s string) Filter {
	return func(a Artifact) bool {
		return a.Goarm == s
	}
}

// ByType is a predefined filter that filters by the given type
func ByType(t Type) Filter {
	return func(a Artifact) bool {
		return a.Type == t
	}
}

// Or performs an OR between all given filters
func Or(filters ...Filter) Filter {
	return func(a Artifact) bool {
		for _, f := range filters {
			if f(a) {
				return true
			}
		}
		return false
	}
}

// And performs an AND between all given filters
func And(filters ...Filter) Filter {
	return func(a Artifact) bool {
		for _, f := range filters {
			if !f(a) {
				return false
			}
		}
		return true
	}
}

// Filter filters the artifact list, returning a new instance.
// There are some pre-defined filters but anything of the Type Filter
// is accepted.
// You can compose filters by using the And and Or filters.
func (artifacts *Artifacts) Filter(filter Filter) Artifacts {
	var result = New()
	for _, a := range artifacts.items {
		if filter(a) {
			result.items = append(result.items, a)
		}
	}
	return result
}
