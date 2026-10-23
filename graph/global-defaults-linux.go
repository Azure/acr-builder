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

# Default image aliases, can be used without $ directive in cmd
acr: mcr.microsoft.com/acr/acr-cli:0.21
az: mcr.microsoft.com/acr/azure-cli:a8f7a6c
bash: mcr.microsoft.com/acr/bash:a8f7a6c
curl: mcr.microsoft.com/acr/curl:a8f7a6c
cssc: mcr.microsoft.com/acr/cssc:a8f7a6
`
