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
acr: mcr.microsoft.com/acr/acr-cli:0.19
az: mcr.microsoft.com/acr/azure-cli:46e2782
bash: mcr.microsoft.com/acr/bash:46e2782
curl: mcr.microsoft.com/acr/curl:46e2782
cssc: mcr.microsoft.com/acr/cssc:46e2782
`
