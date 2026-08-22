#!/usr/bin/env sh

set -euo pipefail

main() {
    # Update CA certificates if needed. Rewriting the system trust store
    # requires root; when running unprivileged rely on the certificates
    # baked into the image.
    if [ "$(id -u)" -eq 0 ]; then
        find /usr/local/share/ca-certificates -maxdepth 0 ! -empty -exec update-ca-certificates \;
    fi

    # Start CMD
    exec "$@"
}

main "$@"
