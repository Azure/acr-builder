package graph

const globalDefaultYamlLinux string = `
# Native Task aliases, use as eg. $Commit
ID: "{{.Run.ID}}"
SharedVolume: "{{.Run.SharedVolume}}"
Registry: "{{.Run.Registry}}"
RegistryName: "{{.Run.RegistryName}}"
Date: "{{.Run.Date}}"
OS: "{{.Run.OS}}"
Architecture: "{{.Run.Architecture}}"
Commit: "{{.Run.Commit}}"
Branch: "{{.Run.Branch}}"

# Usable without $
# Containerized commands can be used without $ directive
purge: "mcr.microsoft.com/acr/acr-cli:0.1 purge"

# Default image aliases, can be used without $ directive
acr: "mcr.microsoft.com/acr/acr-cli:0.1"
az: mcr.microsoft.com/acr/azure-cli:d0725bc
bash: mcr.microsoft.com/acr/bash:d0725bc
curl: mcr.microsoft.com/acr/curl:d0725bc
`
