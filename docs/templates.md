# Templates

Individual builds and pipelines both support templating. Internally we use [Go templates](https://golang.org/pkg/text/template/) and [Sprig](https://github.com/Masterminds/sprig/).

## Using custom values

A sample values file looks like this:

```toml
born = 1867
first = "Marie"
last = "Curie"
research = "radioactivity"
from = "Poland"
```

When this file is loaded with `--values`, you can reference any of the values using this syntax: `{{ .Values.born }}`, `{{ .Values.research }}`, etc.
You could also override any of these values using `--set born=1900` for example.

## Build variables

The following variables can be accessed using `{{ .Build.VariableName }}`, where `VariableName` equals one of the following:

- ID
- Commit
- Repository
- Branch
- TriggeredBy
- Registry
- GitTag
- Date

## Pipelines

You can execute a pipeline with `exec`. It requires a `--steps` file and optionally a `--values` file. You can also use `--set` to override values specified in `--values`, or to provide new values not covered by `--values`.

```bash
$ acb exec --steps templating/testdata/helloworld/git-build.toml --values templating/testdata/helloworld/values.toml --id demo

...

Successfully tagged acr-builder:demo
```

## Build

Templating in `build` works the same as `exec`, except that you don't have to provide a `--steps` file.

```bash
$ acb build https://github.com/Azure/acr-builder.git -f Dockerfile -t "acr-builder:{{.Build.ID}}" --id demo

...

Successfully tagged acr-builder:demo
```