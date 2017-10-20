package tarball

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/packer/common"
	"github.com/mitchellh/packer/helper/config"
	"github.com/mitchellh/packer/packer"
	"github.com/mitchellh/packer/template/interpolate"
)

// Guestfish commands to make required character devices and directories for
// a container or virtual machine.
const makeCharDevices = `mknod-c 0444 1 8 /dev/random
mknod-c 0444 1 9 /dev/urandom
mknod-c 0666 5 0 /dev/tty
mknod-c 0600 5 1 /dev/console
mknod-c 0666 5 2 /dev/ptmx
mknod-c 0666 1 5 /dev/zero
mknod-c 0666 1 3 /dev/null
mkdir-mode /dev/pts 0755
mkdir-mode /dev/shm 0755
`

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	OutputPath            string `mapstructure:"output"`
	TarballFile           string `mapstructure:"tarball_filename"`
	TarballExtension      string `mapstructure:"tarball_extension"`
	GuestfishBinary       string `mapstructure:"guestfish_binary"`
	GuestfishMountTimeout int    `mapstructure:"guestfish_mount_timeout"`
	KeepInputArtifact     bool   `mapstructure:"keep_input_artifact"`
	Compression           string `mapstructure:"compression"`

	ctx interpolate.Context
}

type PostProcessor struct {
	config Config
}

// Result is returned on the channel which waits for output from the stdout
// pipe to Guestfish.
type Result struct {
	Value string
	Err   error
}

func (p *PostProcessor) Configure(raws ...interface{}) error {
	err := config.Decode(&p.config, &config.DecodeOpts{
		Interpolate:        true,
		InterpolateContext: &p.config.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{"output"},
		},
	}, raws...)
	if err != nil {
		return err
	}

	errs := new(packer.MultiError)

	if p.config.GuestfishBinary == "" {
		p.config.GuestfishBinary = "guestfish"
	}

	if p.config.OutputPath == "" {
		p.config.OutputPath = "packer_{{.BuildName}}_tarball"
	}
	if p.config.Compression == "" {
		p.config.Compression = "gz"
	}
	if p.config.TarballExtension == "" {
		p.config.TarballExtension = ".tar.gz"
	}

	if p.config.GuestfishMountTimeout == 0 {
		p.config.GuestfishMountTimeout = 10
	}

	if _, err := exec.LookPath(p.config.GuestfishBinary); err != nil {
		errs = packer.MultiErrorAppend(
			errs, fmt.Errorf("Error finding executable %s: %s", p.config.GuestfishBinary, err))
	}

	if err = interpolate.Validate(p.config.OutputPath, &p.config.ctx); err != nil {
		errs = packer.MultiErrorAppend(
			errs, fmt.Errorf("Error parsing target template: %s", err))
	}

	if len(errs.Errors) > 0 {
		return errs
	}

	return nil
}

func (p *PostProcessor) PostProcess(ui packer.Ui, artifact packer.Artifact) (packer.Artifact, bool, error) {
	// These are extra variables that will be made available for interpolation.
	p.config.ctx.Data = map[string]string{
		"BuildName":   p.config.PackerBuildName,
		"BuilderType": p.config.PackerBuilderType,
	}

	if artifact.BuilderId() != "transcend.qemu" {
		return nil, false, fmt.Errorf("tarball post-processor can only be used with Qemu builder: %s", artifact.BuilderId())
	}

	outputPath, err := interpolate.Render(p.config.OutputPath, &p.config.ctx)
	if err != nil {
		return nil, false, fmt.Errorf("Error interpolating output path: %s", err)
	}

	if _, err := os.Stat(outputPath); err == nil {
		return nil, false, fmt.Errorf("Output directory %s already exists", outputPath)
	}

	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return nil, false, fmt.Errorf("Error creating output directory %s: %s", outputPath, err)
	}

	keep := p.config.KeepInputArtifact
	newArtifact := &Artifact{Path: outputPath}

	for _, src := range artifact.Files() {
		var outfile string

		if p.config.TarballFile == "" {
			outfile = filepath.Join(newArtifact.Path, filepath.Base(src))
		} else {
			outfile = filepath.Join(newArtifact.Path, p.config.TarballFile)
		}

		timeout := p.config.GuestfishMountTimeout
		outfile += p.config.TarballExtension

		gf := exec.Command(p.config.GuestfishBinary)
		w, _ := gf.StdinPipe()
		r, _ := gf.StdoutPipe()
		br := bufio.NewReader(r)
		defer w.Close()
		defer r.Close()

		if err := gf.Start(); err != nil {
			return nil, false, fmt.Errorf("Error running guestfish: %s", err)
		}

		ui.Say(fmt.Sprintf("Loading %s into guestfish", src))
		io.WriteString(w, fmt.Sprintf("add-drive %s\n", src))
		io.WriteString(w, "run\n")
		ui.Message("Finding root filesystem")
		io.WriteString(w, "inspect-os\n")

		// Read the response from Guestfish in a goroutine so that we can timeout
		// if it is having problems finding the root filesystem.
		var result Result
		input := make(chan Result)
		go func(chan Result) {
			line, err := br.ReadString('\n')
			input <- Result{Value: line, Err: err}
		}(input)

		select {
		case result = <-input:
			if result.Err != nil && result.Err != io.EOF {
				return nil, false, fmt.Errorf("Failed to locate root filesystem: %s", err)
			}
		case <-time.After(time.Second * time.Duration(timeout)):
			return nil, false, fmt.Errorf("Failed to locate root filesystem: timed out waiting for response from Guestfish.")
		}

		device := strings.TrimSpace(result.Value)
		ui.Message(fmt.Sprintf("Found root filesystem at %s", device))

		ui.Message(fmt.Sprintf("Mounting %s to /", device))
		io.WriteString(w, fmt.Sprintf("mount %s /\n", device))

		ui.Message("Creating character devices")
		io.WriteString(w, makeCharDevices)

		ui.Message(fmt.Sprintf("Packing filesystem into tarball %s", outfile))
		io.WriteString(w, fmt.Sprintf("tar-out / %s compress:%s\n", outfile, p.config.Compression))
		io.WriteString(w, "quit\n")

		newArtifact.files = append(newArtifact.files, outfile)
	}

	return newArtifact, keep, nil
}
