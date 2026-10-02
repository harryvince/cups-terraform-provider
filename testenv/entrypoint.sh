#!/bin/sh
set -eu

case "${1:-}" in
  cups)
    : "${CUPS_TEST_ADMIN_PASSWORD:?Set a disposable test admin password}"
    printf 'cups-admin:%s\n' "$CUPS_TEST_ADMIN_PASSWORD" | chpasswd
    mkdir -p /run/cups /var/spool/cups /var/cache/cups
    cupsd -t
    exec cupsd -f
    ;;
  printer)
    # CUPS 2.4 initializes Avahi even with advertisements disabled (-r off).
    # These daemons use only the container's private sockets and network.
    mkdir -p /run/dbus
    dbus-daemon --system --fork --nopidfile
    avahi-daemon --daemonize --no-drop-root --no-chroot
    mkdir -p /tmp/ipp-printer
    exec ippeveprinter -r off -p 8000 -d /tmp/ipp-printer \
      -f image/pwg-raster,image/urf,application/pdf -2 -v "Provider Test Printer"
    ;;
  *)
    exec "$@"
    ;;
esac
