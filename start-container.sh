#!/usr/bin/env bash

if [ "$SUPERVISOR_GO_USER" != "root" ] && [ "$SUPERVISOR_GO_USER" != "app" ]; then
    echo "You should set SUPERVISOR_GO_USER to either 'app' or 'root'."
    exit 1
fi

if [ ! -z "$WWWUSER" ]; then
    usermod -u $WWWUSER app
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