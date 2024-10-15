# Novel VMP (temp name)

## About

Novel VMP (temporary name) is a Vulnerability Management Platform developed over the course of my masters thesis that combines the advantages of existing VMPs with the advantages of scanner orchestrators.

## Requirements

- **Go**
- **Docker:** With the user having the rights to interact with the deamon.
- **Linux:** This project was only tested on a Debian Bookworm and NixOS Linux distribution
- **just:** A modern makefile replacement
- **Lots of RAM:** The exact RAM requirements vary on the number of parallel scanners you want to run, but at least 32GB RAM are neccessarry.

## How to Build and Run

```shell
# configure the elk-stack (only needed the first time)
just setup-elk

# start the permanent storage and visualisation (elk-stack)
just elk

# start openvas (required for the openvas scanner backend)
just openvas # this takes around 20 minutes

# build all scanner backends according to the configuration in `orchestrator/config.yaml`
just build-scanners

# Configure the scan!
# - set targets in `orchestrator/config.yaml`
# - configure how many scanners you want to use in parallel: `orchestrator/scanners/[scanner name]/config.yaml`

# start the scan
just run
```

## A Word of Warning

Don't expose anything from this project at its current state to the internet unless you **realy** know what you're doing. The OpenVAS and ELK services are mostly in their default configuration and thus have their default passwords. Also, the orchestrator and scanners uses various ports > 10000, which can be used to remote controll the scanners.