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
acr: mcr.microsoft.com/acr/acr-cli:0.11
az: mcr.microsoft.com/acr/azure-cli:56f0765
bash: mcr.microsoft.com/acr/bash:56f0765
curl: mcr.microsoft.com/acr/curl:56f0765
cssc: mcr.microsoft.com/acr/cssc:56f0765
`
