# Apache HTTP Server Cloud Native Buildpack

The HTTPD CNB provides the [Apache HTTP Server](https://httpd.apache.org) binary distribution.
The buildpack installs the HTTPD binary distribution onto the `$PATH` which makes it available for
subsequent buildpacks, and sets up the start command which signals httpd to start.

A usage example can be found in the
[`samples` repository under the `httpd` directory](https://github.com/paketo-buildpacks/samples/tree/main/httpd).

#### The HTTPD buildpack is compatible with the following builder(s):
- [Paketo Full Builder](https://github.com/paketo-buildpacks/full-builder)

## Integration

The Apache HTTPD CNB provides httpd as a dependency. Downstream buildpacks, like
[php-web](https://github.com/paketo-buildpacks/php-web) can require the httpd
dependency by generating a [Build Plan
TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the Apache HTTPD dependency is "httpd". This value is considered
  # part of the public API for the buildpack and will not change without a plan
  # for deprecation.
  name = "httpd"

  # The version of the Apache HTTPD dependency is not required. In the case it
  # is not specified, the buildpack will provide the newest version, which can
  # be seen in the buildpack.toml file.
  # If you wish to request a specific version, the buildpack supports
  # specifying a semver constraint in the form of "2.*", "2.4.*", or even
  # "2.4.43".
  version = "2.4.43"

  # The Apache HTTPD buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the launch flag to true will ensure that the Apache HTTPD
    # dependency is available on the $PATH for the running application. If you
    # are writing an application that needs to run Apache HTTPD at runtime, this
    # flag should be set to true.
    launch = true
```

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh -v <version>
```

## Configurations

Specifying the HTTP Server version through `buildpack.yml` configuration
will be deprecated in Apache HTTP Server Buildpack v1.0.0.

To migrate from using `buildpack.yml` please set the following environment
variables at build time either directly (ex. `pack build my-app --env
BP_ENVIRONMENT_VARIABLE=some-value`) or through a [`project.toml`
file](https://github.com/buildpacks/spec/blob/main/extensions/project-descriptor.md)

### `BP_HTTPD_VERSION`
The `BP_HTTPD_VERSION` variable allows you to specify the version of Apache HTTP Server that is installed.

```shell
BP_HTTPD_VERSION=2.4.43
```

This will replace the following structure in `buildpack.yml`:
```yaml
httpd:
  # this allows you to specify a version constraint for the httpd dependency
  # any valid semver constraints (e.g. 2.* and 2.4.*) are also acceptable
  version: "2.4.43"
```

## Zero Configuration Variables

The Apache HTTPD Server Buildpack now supports the ability for a user to just
provide a static application without having a `httpd.conf` and the buildpack
will now generate a default `httpd.conf`. In order to activate this workflow
the `BP_WEB_SERVER` environment variable must be set to `httpd`.

```shell
BP_WEB_SERVER=httpd
```

While this will provide a default configuration there are a few modifications
that can be made to this `httpd.conf` by setting the following environment
variables and service bindings.

### `BP_WEB_SERVER_ROOT`
The `BP_WEB_SERVER_ROOT` variable allows you to modify the location of the static files
served by the web server by assigning the `BP_WEB_SERVER_ROOT` variable with an
absolute file path or a file path relative to `/workspace`. For example,
setting `BP_WEB_SERVER_ROOT=my-build-directory` would change the file path of
served files to `/workspace/my-build-directory`.

```shell
BP_WEB_SERVER_ROOT=htdocs
```

### `BP_WEB_SERVER_ENABLE_PUSH_STATE`
The `BP_WEB_SERVER_ENABLE_PUSH_STATE` variable to enable push state routing functionality.

```shell
BP_WEB_SERVER_ENABLE_PUSH_STATE=true
```

### `BP_WEB_SERVE_FORCE_HTTPS`
The `BP_WEB_SERVE_FORCE_HTTPS` variable allows to enforce HTTPS for server connnections.

```shell
BP_WEB_SERVER_FORCE_HTTPS=true
```

### Basic Authentication
You are able to provide basic authentication credentials via an `htpasswd` type
service binding specifying the contents of a `.htpasswd` file. The service
binding will have the following directory structure.

```plain
binding
└── type
└── .htpasswd
```

## Stack support

The HTTPD buildpack requires that you use the Paketo [Full
builder](https://paketo.io/docs/builders/#full) to build applications. The
buildpack does not run on the Base builder because it requires `libexpat1`
that's not present on the Base stack.
