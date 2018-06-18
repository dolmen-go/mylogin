# mylogin - Go utilities for reading MySQL credentials from `~/.mylogin.cnf`

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/dolmen-go/mylogin)
[![Travis-CI](https://img.shields.io/travis/dolmen-go/mylogin.svg)](https://travis-ci.org/dolmen-go/mylogin)
[![Go Report Card](https://goreportcard.com/badge/github.com/dolmen-go/mylogin)](https://goreportcard.com/report/github.com/dolmen-go/mylogin)

About `mylogin.cnf`:

- <https://dev.mysql.com/doc/refman/8.0/en/mysql-config-editor.html>
- <https://dev.mysql.com/doc/mysql-utilities/1.5/en/mysql-utils-intro-connspec-mylogin.cnf.html>

## Go package

[`github.com/dolmen-go/mylogin`](https://godoc.org/github.com/dolmen-go/mylogin) Library for reading and writing `~/.mylogin.cnf`.


## Utilities

### [`mylogin`](https://godoc.org/github.com/dolmen-go/mylogin/cmd/mylogin): dump `~/.mylogin.cnf` content in clear

```sh
go get -u github.com/dolmen-go/cmd/mylogin
mylogin -h
```

### [`mylogin-dsn`](https://godoc.org/github.com/dolmen-go/mylogin/cmd/mylogin-dsn): dump a `mylogin.cnf` section as a [`go-sql-driver/mysql`](https://github.com/go-sql-driver/mysql) connection string prefix

```sh
go get -u github.com/dolmen-go/cmd/mylogin-dsn
mylogin-dsn -h
```

## See also

Package [`github.com/dolmen-go/mylogin-driver/register`](https://godoc.org/github.com/dolmen-go/mylogin-driver/register)
wraps [`github.com/go-sql-driver/mysql`](https://godoc.org/github.com/go-sql-driver/mysql)
with an alternate connection string syntax that allows to refers to a `~/.mylogin.cnf` section.

## License

Copyright 2016-2018 Olivier Mengué

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
