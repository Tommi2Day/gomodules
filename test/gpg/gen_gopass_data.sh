#!/bin/bash
set -uo pipefail

# variables and commands
ME=$(realpath "$0")
WD=$(dirname "$ME")
GPG=$(command -v gpg)
if [ -z "$GPG" ]; then
    echo "gpg not found"
    exit 1
fi

GPG_PRESET="$(ls /usr/lib/gnupg*/gpg-preset-passphrase*)"
if [ -z "$GPG_PRESET" ]; then
    GPG_PRESET=$(command -v gpg-preset-passphrase)
fi
if [ ! -x "$GPG_PRESET" ]; then
    echo "gpg-preset-passphrase not found"
    exit 1
fi
GOPASS=$(command -v gopass)
if [ -z "$GOPASS" ]; then
   go install github.com/gopasspw/gopass@latest
   GOPASS=$(command -v gopass)
fi

set -e


TESTDIR="$WD/.."
KEYDIR="$TESTDIR/gpg"
cd "$WD"
APP="test"
export GNUPGHOME=$TESTDIR/testdata/.gnupg
#export GPG_TTY=/dev/tty
if [ -d "$GNUPGHOME" ]; then
    rm -rf "$GNUPGHOME"
fi


STORE="gopass-store"
STOREPATH="$TESTDIR/$STORE"

PWFILE="$KEYDIR/$APP.gpgpw"
IDFILE="$KEYDIR/$APP.gpgid"
PRIVFILE="$KEYDIR/$APP.priv.gpg"
PUBFILE="$KEYDIR/$APP.pub.gpg"

# generate key
$GPG --batch --gen-key "$APP.keygen"
grep "^Passphrase:" $APP.keygen|sed -e 's/^Passphrase:\s*//g' |tr -d "\r\n">"$PWFILE"
EMAIL=$(grep "^Name-Email:" $APP.keygen|sed -e 's/^Name-Email:\s*//g'|tr -d "\r\n")

# prepare agent
echo "default-cache-ttl 46000
allow-preset-passphrase
" >"$GNUPGHOME/gpg-agent.conf"
gpg-connect-agent reloadagent /bye


$GPG --list-secret-keys

GPG_PASSPHRASE=$(< "$PWFILE" )
KEYID="$(gpg --list-secret-keys |grep "$EMAIL" -B 1|grep '^ '|perl -pe 's/.*\s+(\w+)$/\1/g;')"
KEYGRIP="$(gpg --list-secret-keys --with-keygrip|grep "$EMAIL" -B 1| grep -i 'Keygrip'|perl -pe 's/.*=\s+(\w+)$/\1/g;')"

echo -n "$KEYID" >"$IDFILE"
$GPG --export --armor "$KEYID" >"$PUBFILE"
echo ""
$GPG --pinentry-mode loopback --batch --export-secret-key --armor --passphrase "$GPG_PASSPHRASE" "$KEYID" >"$PRIVFILE"
if [ -x "$GPG_PRESET" ]; then
  "$GPG_PRESET" --preset "$KEYGRIP" <<<"$GPG_PASSPHRASE"
fi


# cleanup
if [ -d "$STOREPATH" ]; then
    rm -rf "$STOREPATH"
fi

# setup gopass if needed
if [ ! -d "$HOME/.config/gopass" ];   then
  $GOPASS setup --email="$EMAIL"  --remote "" --crypto gpgcli --storage fs
fi
# init store

$GOPASS mounts rm "$STORE"
rm -rf "$STOREPATH"
$GOPASS init --path "$STOREPATH" --storage fs --store "$STORE" --crypto gpgcli "$KEYID"

$GOPASS insert "$STORE/passphrase" <"$PWFILE"
$GOPASS insert "$STORE/$APP/test1" <<<"123456"
$GOPASS generate "$STORE/$APP/test2" 16
# test reading
# $GOPASS cat "$STORE/$APP/test2"
