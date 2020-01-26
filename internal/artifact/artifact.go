// Package artifact provides the core artifact storage for goreleaser
package artifact

// nolint: gosec
import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"os"
	"sync"

	"github.com/apex/log"
	"github.com/pkg/errors"
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
	// LinuxPackage is a linux package generated by nfpm
	LinuxPackage
	// PublishableSnapcraft is a snap package yet to be published
	PublishableSnapcraft
	// Snapcraft is a published snap package
	Snapcraft
	// PublishableDockerImage is a Docker image yet to be published
	PublishableDockerImage
	// DockerImage is a published Docker image
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
	case Binary:
		return "Binary"
	case LinuxPackage:
		return "Linux Package"
	case DockerImage:
	case PublishableDockerImage:
		return "Docker Image"
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
	Gomips string
	Type   Type
	Extra  map[string]interface{}
}

// ExtraOr returns the Extra field with the given key or the or value specified
// if it is nil.
func (a Artifact) ExtraOr(key string, or interface{}) interface{} {
	if a.Extra[key] == nil {
		return or
	}
	return a.Extra[key]
}

// Checksum calculates the checksum of the artifact.
// nolint: gosec
func (a Artifact) Checksum(algorithm string) (string, error) {
	log.Debugf("calculating checksum for %s", a.Path)
	file, err := os.Open(a.Path)
	if err != nil {
		return "", errors.Wrap(err, "failed to checksum")
	}
	defer file.Close() // nolint: errcheck
	var h hash.Hash
	switch algorithm {
	case "crc32":
		h = crc32.NewIEEE()
	case "md5":
		h = md5.New()
	case "sha224":
		h = sha256.New224()
	case "sha384":
		h = sha512.New384()
	case "sha256":
		h = sha256.New()
	case "sha1":
		h = sha1.New()
	case "sha512":
		h = sha512.New()
	default:
		return "", fmt.Errorf("invalid algorith: %s", algorithm)
	}
	_, err = io.Copy(h, file)
	if err != nil {
		return "", errors.Wrap(err, "failed to checksum")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Artifacts is a list of artifacts
type Artifacts struct {
	items []*Artifact
	lock  *sync.Mutex
}

// New return a new list of artifacts
func New() Artifacts {
	return Artifacts{
		items: []*Artifact{},
		lock:  &sync.Mutex{},
	}
}

// List return the actual list of artifacts
func (artifacts Artifacts) List() []*Artifact {
	return artifacts.items
}

// GroupByPlatform groups the artifacts by their platform
func (artifacts Artifacts) GroupByPlatform() map[string][]*Artifact {
	var result = map[string][]*Artifact{}
	for _, a := range artifacts.items {
		plat := a.Goos + a.Goarch + a.Goarm
		result[plat] = append(result[plat], a)
	}
	return result
}

// Add safely adds a new artifact to an artifact list
func (artifacts *Artifacts) Add(a *Artifact) {
	artifacts.lock.Lock()
	defer artifacts.lock.Unlock()
	log.WithFields(log.Fields{
		"name": a.Name,
		"path": a.Path,
		"type": a.Type,
	}).Debug("added new artifact")
	artifacts.items = append(artifacts.items, a)
}

// Filter defines an artifact filter which can be used within the Filter
// function
type Filter func(a *Artifact) bool

// ByGoos is a predefined filter that filters by the given goos
func ByGoos(s string) Filter {
	return func(a *Artifact) bool {
		return a.Goos == s
	}
}

// ByGoarch is a predefined filter that filters by the given goarch
func ByGoarch(s string) Filter {
	return func(a *Artifact) bool {
		return a.Goarch == s
	}
}

// ByGoarm is a predefined filter that filters by the given goarm
func ByGoarm(s string) Filter {
	return func(a *Artifact) bool {
		return a.Goarm == s
	}
}

// ByType is a predefined filter that filters by the given type
func ByType(t Type) Filter {
	return func(a *Artifact) bool {
		return a.Type == t
	}
}

// ByFormats filters artifacts by a `Format` extra field.
func ByFormats(formats ...string) Filter {
	var filters = make([]Filter, 0, len(formats))
	for _, format := range formats {
		format := format
		filters = append(filters, func(a *Artifact) bool {
			return a.ExtraOr("Format", "") == format
		})
	}
	return Or(filters...)
}

// ByIDs filter artifacts by an `ID` extra field.
func ByIDs(ids ...string) Filter {
	var filters = make([]Filter, 0, len(ids))
	for _, id := range ids {
		id := id
		filters = append(filters, func(a *Artifact) bool {
			// checksum are allways for all artifacts, so return always true.
			return a.Type == Checksum || a.ExtraOr("ID", "") == id
		})
	}
	return Or(filters...)
}

// Or performs an OR between all given filters
func Or(filters ...Filter) Filter {
	return func(a *Artifact) bool {
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
	return func(a *Artifact) bool {
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
