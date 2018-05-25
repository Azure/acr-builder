# Templates

### A build can have multiple templates located in `templates/`:

```
# acr-builder-pre-release.toml
push = ["Image1", "Image2"]

[[step]]
id = "A"
name = "eric-build"
args = ["build", "-t", "{{ .Build.QueuedBy | upper }}", "."]

[[step]]
name = "eric-build-v2"
id = "B"
whenAll = ["A'] # empty -> no waiting
args = ["build", "-t", "{{ .Build.BuildId | lower }}", "."]
workDir = "subdirectory"
env = ["ENV1=Foo", "ENV2=Bar"]

[[step]]
id = "eric-push"
args = ["push", "azacr.someregistry.io/{{RepoId}}/{{ImageName | default "someImageName"}}"]
```

### Specify values to override in your templates:

```
# values.toml

# Default values
ImageName = "DefaultImageName"
```

```
# release-values.toml

ImageName = "ProdBuild"
```


### Combine the two to create a standardized pipeline:

```
# rally.toml

push = ["Image1", "Image2"]

[[step]]
name = "eric-build"
args = ["build", "-t", "ERIC", "."]
id = "A"

[[step]]
name = "eric-build-v2"
id = "B"
whenAll = ["A"] # empty -> no waiting
args = ["build", "-t", "eus-12345", "."]
workDir = "subdirectory"
env = ["ENV1=Foo", "ENV2=Bar"]

[[step]]
name = "eric-push"
args = ["push", "azacr.someregistry.io/ehotinger/acr-builder"]

```