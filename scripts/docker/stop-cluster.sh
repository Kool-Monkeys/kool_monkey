#!/bin/bash

docker rm -fv $(docker ps -q)
docker rm kool-agent kool-api kool-database postgres-db-data
