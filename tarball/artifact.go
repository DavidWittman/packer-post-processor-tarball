package tarball

import (
	"os"
	"strings"
)

const BuilderId = "DavidWittman.post-processor.tarball"

type Artifact struct {
	Path  string
	files []string
}

func NewArtifact(path string) *Artifact {
	return &Artifact{Path: path}
}

func (a *Artifact) BuilderId() string {
	return BuilderId
}

func (a *Artifact) Id() string {
	return "TARBALL"
}

func (a *Artifact) Files() []string {
	return a.files
}

func (a *Artifact) String() string {
	return strings.Join(a.files, ", ")
}

func (a *Artifact) State(name string) interface{} {
	return nil
}

func (a *Artifact) Destroy() error {
	return os.RemoveAll(a.Path)
}
