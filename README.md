# Tarball Post Processor for Packer

This post-processor for [Packer](https://packer.io) creates a tarball of the root filesystem using [Guestfish](http://libguestfs.org/guestfish.1.html). The resulting image is suitable for deploying OpenVZ containers.

The Qemu builder is used in favor of a writing a separate OpenVZ builder for a few reasons:

 - Preference building from an ISO over an existing template
 - Qemu does not require a proprietary kernel
 - The Qemu builder is stable, and writing post-processors is relatively easy
 - Having a tarball of the filesystem potentially has uses outside of OpenVZ

## Requirements

 - Packer 0.8.x+
 - Guestfish (and libguestfs)
 - Qemu (preferrably with KVM acceleration)

## Installation

First, download and install the plugin:

``` bash
$ go get github.com/DavidWittman/packer-post-processor-tarball
```

Then, move the binary from `$GOPATH/bin/packer-post-processor-tarball` to one of three locations:

 - The directory where packer is, or the executable directory
 - `~/.packer.d/plugins/` (Unix) or `%APPDATA%/packer.d/plugins` (Windows)
 - The current working directory

## Usage

Once the `packer-post-processor-tarball` binary is in a directory which is discoverable by Packer, you can add this post-processor to your template as follows:

``` json
{
  "post-processors": ["tarball"]
}
```

### Configuration

Here are all of the available configuration options (and their defaults) for this post-processor. Keys beginning with an underscore are comments.

``` json
{
  "post-processors": [
    {
      "type": "tarball",

      "_comment": "The directory to write artifacts to",
      "output": "packer_{{.BuildName}}_tarball",

      "_comment": "Filename to use for the artifact. `.tar.gz` will be appended to the end",
      "tarball_filename": "packer_{{.BuildName}}",

      "_comment": "The Guestfish binary to use",
      "guestfish_binary": "guestfish",

      "_comment": "How long (in seconds) to wait for Guestfish to mount the file system",
      "guestfish_root_fs_mount_timeout": 10,

      "_comment": "Keep the input artifact which was received by this post-processor",
      "keep_input_artifact": false
    }
  ]
}
```

## Examples

See the Packer templates at [DavidWittman/packer-openvz-templates](https://github.com/DavidWittman/packer-openvz-templates).
