#!/usr/bin/env bash

if [ "$SUPERVISOR_GO_USER" != "root" ] && [ "$SUPERVISOR_GO_USER" != "app" ]; then
    echo "You should set SUPERVISOR_GO_USER to either 'app' or 'root'."
    exit 1
fi

# The image already creates app with uid WWWUSER, so this is normally a no-op.
# It stays for the case where the runtime WWWUSER differs from the build-time one,
# and is guarded because usermod fails when the uid is already correct.
if [ -n "$WWWUSER" ]; then
    current_uid="$(id -u app)"

    if [ "$current_uid" != "$WWWUSER" ]; then
        usermod -u "$WWWUSER" app
    fi
fi

# Do some staff here

if [ $# -gt 0 ]; then
    if [ "$SUPERVISOR_GO_USER" = "root" ]; then
        exec "$@"
    else
        exec gosu $WWWUSER "$@"
    fi
else
    exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
fi