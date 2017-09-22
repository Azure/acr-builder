package grok

import "github.com/Azure/acr-builder/pkg/domain"

const DotnetExampleTargetRegistryName = "registry"
const DotnetExampleTargetImageName = "img"

var DotnetExampleDependencies = domain.ImageDependencies{
	Image:             DotnetExampleTargetRegistryName + "/" + DotnetExampleTargetImageName,
	RuntimeDependency: "microsoft/aspnetcore:2.0",
	BuildDependencies: []string{"microsoft/aspnetcore-build:2.0", "imaginary/cert-generator:1.0"},
}
