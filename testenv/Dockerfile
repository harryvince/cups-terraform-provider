FROM debian:bookworm-slim

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        cups cups-ipp-utils curl python3 avahi-daemon dbus socat \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --create-home --groups lpadmin cups-admin

COPY cupsd.conf /etc/cups/cupsd.conf
COPY entrypoint.sh /usr/local/bin/testenv-entrypoint
COPY smoke.py /opt/testenv/smoke.py
RUN chmod +x /usr/local/bin/testenv-entrypoint \
    && cupsd -t

ENTRYPOINT ["/usr/local/bin/testenv-entrypoint"]
