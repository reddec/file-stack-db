# file-stack-db
Key with historical values database based on [file-stack](http://github.com/reddec/file-stack)

It provides:

* Dynamic allocation or reusing of file stacks
* Auto closing unused stacks that reduce file handlers usage (especially for low-cost platforms)

# Tools

Stack DB with RPC/RPC-HTTP/HTTP API:

    go get -u github.com/reddec/file-stack-db/cmd/...

Use package manager for Debian/Centos by [packager.io](https://packager.io/gh/reddec/file-stack-db)

# Dev documentation

See [godoc](http://godoc.org/github.com/reddec/file-stack-db)

# HTTP API

See [swagger UI](http://editor.swagger.io/#/?import=https://raw.githubusercontent.com/reddec/file-stack-db/master/swagger.yaml)
or [swagger.yaml](swagger.yaml)
