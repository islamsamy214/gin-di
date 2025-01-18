#!/usr/bin/env bash

if [ "$SUPERVISOR_GO_USER" != "root" ] && [ "$SUPERVISOR_GO_USER" != "app" ]; then
    echo "You should set SUPERVISOR_GO_USER to either 'app' or 'root'."
    exit 1
fi

if [ ! -z "$WWWUSER" ]; then
    usermod -u $WWWUSER app
fi

export PATH=$PATH:/usr/local/go/bin
export GOPATH=/var/www/html/go
export GOMODCACHE=/var/www/html/go/pkg/mod
export GOBIN=/var/www/html/go/bin

go run /var/www/html/app/console/kernel.go migrate

chown -R app:app /var/www/html
chmod -R 755 /var/www/html/storage

if [ $# -gt 0 ]; then
    if [ "$SUPERVISOR_GO_USER" = "root" ]; then
        exec "$@"
    else
        exec gosu $WWWUSER "$@"
    fi
else
    exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
fi