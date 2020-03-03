#!/usr/bin/env bash

FILENAME="hosts_docker.yaml"
STARTPORT=2000
ENDPORT=2049

#docker build -t centos:ssh .
#for i in `seq $STARTPORT $ENDPORT`; do echo $i; done | xargs -n 1 -I % docker run -d -p %:22 centos:ssh
PORTS=`docker ps --format "{{ .Ports }}" | cut -d\: -f2 | cut -d- -f1`
echo "nodes:" > $FILENAME
for p in $PORTS
do
  echo "  - server: \"localhost\"" >> $FILENAME
  echo "    port: \"$p\"" >> $FILENAME
done
echo "defaults:
  port: "22"
  user: "root"
  password: 'cisco'" >> $FILENAME
mv $FILENAME ..