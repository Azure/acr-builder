package grok

import "github.com/Azure/acr-builder/pkg/domain"

// ImageDependenciesResolver resolves docker image dependencies
type ImageDependenciesResolver interface {
	Resolve(domain.Runner) ([]domain.ImageDependencies, error)
}
