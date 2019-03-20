# Templates

Individual builds and tasks both support templating. Internally we use [Go templates](https://golang.org/pkg/text/template/) and [Sprig](https://github.com/Masterminds/sprig/).

## Using custom values

A sample values file looks like this:

```yaml
born: 1867
first: Marie
last: Curie
research: radioactivity
from: Poland
```

When this file is loaded with `--values`, you can reference any of the values using this syntax: `{{ .Values.born }}`, `{{ .Values.research }}`, etc.
You could also override any of these values using `--set born=1900` for example.

## Build variables

The following variables can be accessed using `{{ .Run.VariableName }}`, where `VariableName` equals one of the following:

- `ID`
- `SharedVolume`
- `Registry` (the fully qualified registry name)
- `RegistryName` (just the name of the registry)
- `Date` (`yyyyMMdd-HHmmssz` format)
- `OS`
- `Architecture`

## Tasks

You can execute a task with `exec`. It requires a `-f` file and optionally a `--values` file. You can also use `--set` to override values specified in `--values`, or to provide new values not covered by `--values`.

```sh
$ acb exec -f templating/testdata/helloworld/git-build.yaml --values templating/testdata/helloworld/values.yaml --id demo

...

Successfully tagged acr-builder:demo
```

## Build

Templating in `build` works the same as `exec`, except that you don't have to provide a `Task` file.

```sh
$ acb build https://github.com/Azure/acr-builder.git -f Dockerfile -t "acr-builder:{{.Run.ID}}" --id demo

...

Successfully tagged acr-builder:demo
```