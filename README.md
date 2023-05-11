
# wave

Internal tool to build and deploy applications.


## Install

```shell
go install github.com/altipla-consulting/wave@latest
```


## Build new containers

Build a new container with the application `myname`:

```shell
wave build myname --project google-project-foo
```

Inside our normal Jenkins scripts where a variable is defined to configure gcloud previously:

```shell
wave build myname
```

Dockerfile must be organized inside a folder with the name of the application: `myname/Dockerfile`. Container will build from the directory where this application runs to allow cross-applications package imports.

You can build multiple containers at the same time:

```shell
wave build foo bar baz --project $GOOGLE_PROJECT
```


## Deploy to Cloud Run

Generic execution in any environment:

```shell
wave deploy myname --project google-project-foo --sentry foo-name
```

Inside our normal Jenkins scripts where a variable is defined to configure gcloud previously:

```shell
wave deploy myname --sentry foo-name
```

You can deploy multiple containers at the same time:

```shell
wave deploy foo bar baz --project $GOOGLE_PROJECT --sentry foo-name
```


## Contributing

You can make pull requests or create issues in GitHub. Any code you send should be formatted using `make gofmt`.


## License

[MIT License](LICENSE)
