# Templates

Both builds (`acb build`, `az acr build`) and tasks (`acb exec`, `az acr run -f acb.yaml`) support templating when running the builder locally. However, if the builder is invoked through Azure, only tasks support value-based templating. Builds can only render [run variables](#run-variables).

Internally we use [Go templates](https://golang.org/pkg/text/template/) and [Sprig](https://github.com/Masterminds/sprig/) to perform the rendering. Please review their documentation for a list of all the available template functions.

## Custom values

A `values.yaml` file consists of key/value pairs, such as:

```yaml
born: 1867
first: Marie
last: Curie
research: radioactivity
from: Poland
```

When this file is provided via `--values`, you can reference any of the values using `{{ .Values.born }}`, `{{ .Values.research }}`, etc.

You can override any of these values using `--set key=value`. For example, using `--set born=1900` would cause `{{.Values.born}}` to render as `1900`.

## Run variables

The following variables can be accessed using `{{ .Run.VariableName }}`, where `VariableName` equals one of the following:

| Variable Name | Description |
|---------------|-------------|
| `ID` | The unique identifier of the run |
| `SharedVolume` | The unique identifier of the shared volume, which is accessible by all steps |
| `Registry` | The fully qualified registry name |
| `RegistryName` | The name of the container registry |
| `Date` | The start time of the run in `yyyyMMdd-HHmmssz` format |
| `OS` | The operating system being used |
| `Architecture` | The architecture being used |
| `Commit` | The commit that triggered the run or the latest commit from the actively checked out branch |
| `Branch` | The branch that triggered the run or the branch which is checked out after cloning |
| `TaskName` | The name of the task that triggered this run |

Note that certain properties such as `Commit` and `Branch` will not be available at all times. For example, if you manually queue a run which uploads a context that doesn't contain a `.git` folder.