#!/bin/bash

make docker-database

# Start Database Container
docker create -v /var/lib/postgresql/data --name postgres-db-data postgres:9.5 /bin/true
docker run -d --name kool-database --volumes-from postgres-db-data -e POSTGRES_PASSWORD=${USERNAME} kool-database

make docker-api

# Start API service
docker run -it -d --name kool-api --link kool-database kool-api:latest

make docker-agent

# Start Agent service
docker run -it -d --name kool-agent --link kool-api kool-agent:latest
