<div align="center">
  <a href="https://github.com/OperantAI/woodpecker/actions/workflows/build.yml">
    <img src="https://github.com/OperantAI/woodpecker/actions/workflows/build.yml/badge.svg?branch=main">
  </a>
  <a href="https://github.com/operantai/woodpecker/issues">
    <img src="https://img.shields.io/github/issues/operantai/woodpecker">
  </a>
  <a href ="https://github.com/operantai/woodpecker/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/operantai/woodpecker">
  </a>
</div>
<br />
<div align="center">
  <h3 align="center">woodpecker</h3>
  <p align="center">
    Red-teaming for AI and Cloud
    <br />
    <a href="https://github.com/operantai/woodpecker/blob/main/README.md"><strong>Explore the docs »</strong></a>
    <br />
    <br />
    <a href="https://github.com/operantai/woodpecker/blob/main/CONTRIBUTING.md#reporting-bugs">Report Bug</a>
    ·
    <a href="https://github.com/operantai/woodpecker/blob/main/CONTRIBUTING.md#suggesting-enhancements">Request Feature</a>
  </p>
</div>

**woodpecker** is a modular red teaming tool focused for AI and cloud apps. The tool is designed to discover security weaknesses by experimentation.

## Getting Started

### Installation

You can fetch the latest release [here][latest-release-url], or you can build from source.

#### Building from Source

To build from source, you'll need to have [Go](https://golang.org/) installed.

```sh
git clone https://github.com/operantai/woodpecker
cd woodpecker
make build
```

### Usage

The design of **woodpecker** can be broken down into three concepts:

- **Experiments** - Experiments actively try to run something to discover if a security weakness is present.
- **Verifiers** - Verifiers look at the results of an Experiment and reports their outcome.
- **Components** - Components are additional applications installed on a K8s cluster or in Docker to enable and enhance experiment functionality.

The woodpecker CLI mirrors this, and exposes `experiment`, and `component` commands.

#### Experiments & Verifiers

To start, you need to run an experiment.

Each experiment is defined by a `experiment` file which allows you to tweak your experiment parameters to suit your scenarios.

For a full list of experiments available, you can run `woodpecker experiment` and you'll get a list and a short description of their capabilities.

To get you started you can then run `woodpecker experiment snippet -e <experiment-name>` and it'll output a template you can start from.

Once you're happy with your template you can run it:

``` sh
$ woodpecker experiment run -f experiments/host_path_volume.yaml
```

Once you've successfully run the experiment, you can verify if it was sucessful or not:

```sh
$ woodpecker experiment verify -f experiments/host_path_volume.yaml
```

You can also output in various formats using `-o json` or `-o yaml`

#### Components

Some experiments require additional applications installed to run or enhance their functionality.

These can be added by providing a YAML file, see the [components][components-dir-url] directory for examples.

```sh
$ woodpecker component install -f components/woodpecker-ai.yaml
$ woodpecker component uninstall -f components/woodpecker-ai.yaml
```

Experiments that need a component will warn you if it's not deployed when trying to run it.

## Contributing

Please read the contribution guidelines, [here][contributing-url].

## License

Distributed under the [Apache License 2.0][license-url].

[latest-release-url]: https://github.com/operantai/woodpecker/releases/latest
[experiments-dir-url]: https://github.com/operantai/woodpecker/blob/main/experiments
[components-dir-url]: https://github.com/operantai/woodpecker/blob/main/components
[contributing-url]: https://github.com/operantai/woodpecker/blob/main/CONTRIBUTING.md
[license-url]: https://github.com/operantai/woodpecker/blob/main/LICENSE
