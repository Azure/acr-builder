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
acr: mcr.microsoft.com/acr/acr-cli:0.15
az: mcr.microsoft.com/acr/azure-cli:cbcf692
bash: mcr.microsoft.com/acr/bash:cbcf692
curl: mcr.microsoft.com/acr/curl:cbcf692
cssc: mcr.microsoft.com/acr/cssc:cbcf692
`
