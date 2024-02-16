#!/bin/bash

WD=$(pwd)

NAME=mailserver
DOMAIN=test.local
HOSTNAME=mail

docker rm -f  $NAME
rm -rf config ssl
mkdir config ssl
# '--user' is to keep ownership of the files written to
# the local volume to use your systems User and Group ID values.
docker run --rm -it \
  --user "$(id -u):$(id -g)" \
  --volume "${WD}/gen_certs.sh:/tmp/gen_certs.sh:ro" \
  --volume "${WD}:/tmp/step-ca/" \
  --workdir "/tmp/step-ca/" \
  -e SERVER=$HOSTNAME \
  -e DOMAIN=$DOMAIN \
  -e SSLDIR=ssl \
  --entrypoint "/tmp/gen_certs.sh" \
  smallstep/step-ca


docker run -d \
  --name $NAME \
  --hostname $HOSTNAME \
  --domainname $DOMAIN \
  -e PERMIT_DOCKER=network \
  -e LOG_LEVEL=debug \
  -e ONE_FILE=1 \
  -e TZ=UTC \
  -e ENABLE_OPENDKIM=0 \
  -e ENABLE_OPENDMARC=0 \
  -e ENABLE_AMAVIS=0 \
  -e SSL_TYPE=manual \
  -e SSL_CERT_PATH=/tmp/dms/custom-certs/${HOSTNAME}.${DOMAIN}-full.crt\
  -e SSL_KEY_PATH=/tmp/dms/custom-certs/${HOSTNAME}.${DOMAIN}.key \
  -v $WD/ssl:/tmp/dms/custom-certs/:ro \
  -v $WD/config:/tmp/docker-mailserver/ \
  -p 1025:25 \
  -p 1110:110 \
  -p 1143:143 \
  -p 1465:465 \
  -p 1587:587 \
  -p 1993:993 \
  docker.io/mailserver/docker-mailserver:latest

docker exec -it mailserver setup email add root@$DOMAIN testpass
docker exec -it mailserver setup email add info@$DOMAIN testpass
docker exec -it mailserver setup alias add postmaster@$DOMAIN root@$DOMAIN
