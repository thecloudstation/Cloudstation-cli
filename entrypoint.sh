#!/bin/sh
# Simplified entrypoint - skip Azure authentication in Nomad environment
# Azure login is not required since the check-image task already handles ACR authentication
exec "$@"
