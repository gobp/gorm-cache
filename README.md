# GoBP GORM Cache

GoBP GORM Cache plugin optimize database performance by using response caching mechanism.

## Features

- [ ] Database request reduction. If three identical requests are running at the same time, only the first one is going to be executed, and its response will be returned for all.
- [X] Database response caching. By implementing the Cacher interface, you can easily setup a caching mechanism for your database queries.
- [X] Supports all databases that are supported by gorm itself.


## Install

TBD

## Usage

TBD

## References:

- [gorm.io/caches](https://github.com/go-gorm/caches)
- [liyuan1125/gorm-cache](https://github.com/liyuan1125/gorm-cache)

## Licenses

MIT