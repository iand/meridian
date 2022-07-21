# meridian
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/iand/meridian)

An HTTP Gateway for IPFS


## Overview

This is currently an testbed for gateway experiments.


## Getting Started

As of Go 1.18, install the latest meridian executable using:

	go install github.com/iand/meridian@latest

This will download and build an meridian binary in `$GOBIN`

## Configuration

Meridian is experimenting with the use of a composable module system. 
The granularity of composability is still in flux and some of the current systems may be removed or combined with others.

The module system aims to be composable via configuration and no special build systems other than importing the necessary code.

Modules are organised into categories such as `datastore`, `blockstore` or `routing`. 
A full list of categories can be found in [`conf/module.g`](conf/module.go).

Each module must provide a unique identifier within its category using its `ID` method.
A module registers itself with Meridian by calling `conf.RegisterModule` specifying its category and an instance of the module's type. 
This is normally performed in the module's `init` function so that importing its package will automatically register the module as available for use.
Each category is associated with one or more interfaces that the registered instance must implement. 
For example, the `datastore` category expects modules to implement the `DatastoreProvider` interface.

Meridian uses a single JSON file for configuration.
When a module is referenced by id in the config file Meridian will attempt to load it by populating the registered instance with the config found under that identifier.
The appropriate interface method is then called on the populated instance.

Some systems such as `blockstore` expect only one module to be specified but others may accept several. 
Modules are always processed in the order they are declated within a system's config.
Some systems adapt or enchance others. 
For example the `blockstore_wrapper` system expects a list of modules that can wrap a blockstore to add more functionality such as caching or filtering.
The `routing_composer` system provides a way of configuring how different `routing` modules will be used. such as in parallel or in sequence.

To make a new module available for use in configuration simply import that module's package before calling Meridian's `Main` function.
The [default command line package](cmd/meridian/main.go) imports all the modules declared in this repository so they are all available for use in configuration.













## Contributing

Welcoming [new issues](https://github.com/iand/meridian/issues/new) and [pull requests](https://github.com/iand/meridian/pulls).

## License

This software is dual-licensed under Apache 2.0 and MIT terms:

- Apache License, Version 2.0, ([LICENSE-APACHE](LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](LICENSE-MIT) or http://opensource.org/licenses/MIT)
