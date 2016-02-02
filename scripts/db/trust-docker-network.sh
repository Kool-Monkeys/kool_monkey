#!/bin/bash

PG_HBA_IN="/pg_hba_docker.conf"
PG_HBA_OUT="/var/lib/postgresql/data/pg_hba.conf"

cp ${PG_HBA_IN} ${PG_HBA_OUT}
cat ${PG_HBA_OUT}
