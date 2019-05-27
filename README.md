#nuvoprog - Nuvoton microcontroller programmer

`nuvoprog` is an open source tool for programming Nuvoton microcontollers;
previously, they could only be programmed under Windows using Nuvoton's
proprietary tools. This tool assumes a Nuvoton NuLink family programmer
or compatible; no support is provided (yet) for other programmers.

This tool should be reasonably robust but presently has very limited device
support; if you wish to add support for new devices, that would be much
appreciated. Information on how to do so is at the bottom of this readme

Additionally, a human-friendly (JSON) interface to the configuration bits
is provided.

The tool provides integrated help

Example usage:
```
$ nuvoprog read -t n76e003 dev.ihx
$ nuvoprog config decode -i dev.ihx
$ nuvoprog program -t n76e003 -c @config.json -a aprom.ihx -l ldrom.ihx

```

You may also be interested in [libn76](https://github.com/erincandescent/libn76),
a SDCC-supporting BSP for the Nuvoton N76 family.

*Cortex-M devices*: While I have no objections to someone adding support for
these, have you considered OpenOCD?

# Installing
This is a Go project; install a Go toolchain and install it
using `go get -u github.com/erincandescent/nuvoprog`. Ensure
that `$GOPATH/bin` is on your path (`GOPATH` defaults to `$HOME/go`);
alternatively, move the resulting binary to a location of your choice.

The `hidapi` and `libusb` packages are [vendored by our upstream](https://github.com/karalabe/hid)

# Supported Devices
## Programmers

 *  Nu-Link-Me (as found on Nu-Tiny devboards)

Coming soon:

 * Nu-Link

## Target devices

 * N76E003 (8051T1 family)

# Missing functionality

* Firmware upgrades
* Debugging?

# Adding support for new devices

To add support for new devices, you will need:

 * Windows
 * The Nuvoton ICP tool, and
 * Wireshark

A Wireshark dissector for the protocol can be found in the misc directory.

Nuvoton have [an OpenOCD patch](http://openocd.zylin.com/#/c/4739/1) which you may find useful as reference material

## Other NuLink Programmers
If this is a protocol v2 programmer, you'll need to add support for that (The leading length field
changes from 8 to 16 bits, but othewise things are unchanged).

Add the VID and PID to the table in `protocol/device.go` and see if `nuvoprog` connects successfully.
If it doesn't, compare protocol exchanges in Wireshark

## Other Microcontrollers
First step is to see if the microcontroller belongs to the same family and if the connection and
programming flow is the same (The flow should be the same for the 8051T1 family, may differ for
others).

If they are, you probably just need to define target devide details:

 * Configuration bit codec
 * Target definition (see `target/n76/n76e003.go`)

You may need to get details like LDROM offsets from Wireshark dumps
