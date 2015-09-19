# Tarball Post Processor for Packer

This post-processor for [Packer](https://packer.io) creates a tarball of the root filesystem using Guestfish. The resulting image is suitable for deploying OpenVZ containers.

## Requirements

 - Packer 0.8.x+
 - Guestfish (and libguestfs)
 - Qemu (preferrably with KVM acceleration)

## Installation

TODO

## Usage

Once the `packer-post-processor-tarball` binary is in a directory which is discoverable by Packer, you can add this post-processor to your template as follows:

``` json
{
  "post-processors": [
    {
      "type": "tarball"
    }
  ]
}
```

### Configurables

TODO

## Examples

TODO
