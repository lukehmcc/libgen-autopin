# libgen-autopin

Utility for auto-pinning libgen on IPFS.

![](static/demo.cast.svg)

### Overview

Currently, repinning libgen on IPFS is a tedious and time-consuming task. This tool just automates scraping the CIDs from [freeread](https://freeread.org/), and then repins them for you.

### Installation

**Local Build**:

```
git clone https://github.com/lukehmcc/libgen-autopin.git
go build
```

**Install through go**:

````
go install github.com/lukehmcc/libgen-autopin/libgen-autopin@latest # or target a specific version```
````

### Usage

```
Welcome to libgen-autopin!
NAME:
   libgen-autopin - easily re-pin libgen on IPFS

USAGE:
   libgen-autopin [optional flags]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --quota value, -q value  Storage quota allocated for pinning (GB) (default: 50)
   --node value, -n value   IPFS Node (default: "http://127.0.0.1:5001")
   --help, -h               show help
```
