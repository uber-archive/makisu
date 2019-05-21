#! /bin/bash

set -e

[ -z "$1" ] && echo "Must at least provide context of build" && exit 1

CHROOT=${CHROOT_LOCATION:-$HOME/.makisu-chroot-$RANDOM}
SSL_CERT_DIR=${SSL_CERTS:-/etc/ssl/certs}
CONTEXT=${@: -1}
BUILD_VOLUMES="$CONTEXT:/context,$BUILD_VOLUMES"

function makisu::prepare_internals () {
    mkdir -p $CHROOT/makisu-internal/certs
    cp $(which makisu) $CHROOT/makisu-internal/makisu
    cat $SSL_CERT_DIR/* > $CHROOT/makisu-internal/certs/cacerts.pem
}

function makisu::prepare_dev () {
    mkdir -p $CHROOT/dev
    mknod -m 622 $CHROOT/dev/console c 5 1
    mknod -m 666 $CHROOT/dev/null c 1 3
    mknod -m 666 $CHROOT/dev/zero c 1 5
    mknod -m 666 $CHROOT/dev/ptmx c 5 2
    mknod -m 666 $CHROOT/dev/tty c 5 0
    mknod -m 444 $CHROOT/dev/random c 1 8
    mknod -m 444 $CHROOT/dev/urandom c 1 9
    chown root:tty $CHROOT/dev/{console,ptmx,tty}
}

function makisu::prepare_etc () {
    mkdir -p $CHROOT/etc
    cp /etc/*.conf $CHROOT/etc/
}

function makisu::prepare_volumes () {
    for vol in $(sed "s/,/ /g" <<< $BUILD_VOLUMES); do
        from=$(cut -d ':' -f 1 <<< $vol)
        to=$(cut -d ':' -f 2 <<< $vol)
        echo "Copying volume $from to chroot directory $CHROOT/$to"
        mkdir -p $CHROOT/$to
        cp -r $from/* $CHROOT/$to
    done
}

echo "Preparing chroot at $CHROOT"
rm -rf $CHROOT

makisu::prepare_internals
makisu::prepare_etc
makisu::prepare_dev
makisu::prepare_volumes

makisu_args=${@:1:$#-1}
echo "Starting Makisu: makisu $makisu_args /context"
chroot $CHROOT/ /makisu-internal/makisu $makisu_args /context
