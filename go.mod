module github.com/Azure/acr-builder

replace github.com/docker/docker => github.com/docker/docker v0.0.0-20180413235638-61138fb5fc0f

require (
	github.com/Azure/azure-sdk-for-go v31.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.2
	github.com/Azure/go-autorest/autorest/adal v0.9.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.0
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.0 // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.20.0+incompatible
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/containerd/containerd v1.3.6
	github.com/deislabs/oras v0.8.1
	github.com/docker/cli v0.0.0-20200130152716-5d0cf8839492
	github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/gogo/googleapis v1.4.0 // indirect
	github.com/google/uuid v0.0.0-20161128191214-064e2069ce9c
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.5 // indirect
	github.com/opencontainers/selinux v1.0.0-rc1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/urfave/cli v1.21.0
	golang.org/x/time v0.0.0-20180412165947-fbb02b2291d2 // indirect
	gopkg.in/yaml.v2 v2.2.2
)

go 1.13
