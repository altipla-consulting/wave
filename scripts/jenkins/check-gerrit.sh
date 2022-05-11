#!/bin/bash

set -eux

make build
make lint
make test
