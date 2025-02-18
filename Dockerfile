FROM ubuntu:24.04

LABEL maintainer="Islam Samy"

# Arguments
ARG WWWGROUP=1000
ARG WWWUSER=1000
ARG GOLANG_VERSION=1.23.5
# ARG MYSQL_CLIENT="mysql-client"
ARG POSTGRES_VERSION=17

# Workdir
WORKDIR /web-app

# Environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=UTC
ENV SUPERVISOR_GO_COMMAND="/opt/go-bins/main http"
ENV SUPERVISOR_GO_USER="app"
ENV PGSSLCERT /tmp/postgresql.crt

# Define the timezone
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# Set apt-get noninteractive
RUN echo "Acquire::http::Pipeline-Depth 0;" > /etc/apt/apt.conf.d/99custom && \
    echo "Acquire::http::No-Cache true;" >> /etc/apt/apt.conf.d/99custom && \
    echo "Acquire::BrokenProxy    true;" >> /etc/apt/apt.conf.d/99custom

# Install dependencies
RUN apt-get update && apt-get upgrade -y \
    && mkdir -p /etc/apt/keyrings \
    && apt-get install -y gnupg gosu curl ca-certificates zip unzip git supervisor sqlite3 libcap2-bin libpng-dev python3 dnsutils librsvg2-bin fswatch ffmpeg nano vim librdkafka-dev

# Install golang
RUN apt-get update && apt-get install -y wget && \
    wget https://go.dev/dl/go$GOLANG_VERSION.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go$GOLANG_VERSION.linux-amd64.tar.gz && \
    rm go$GOLANG_VERSION.linux-amd64.tar.gz && \
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile

# Set the go environment variables
ENV PATH=$PATH:/usr/local/go/bin
ENV GOCACHE=/var/tmp/go-cache
ENV GOPATH=/web-app/go
ENV GOMODCACHE=/web-app/go/pkg/mod
ENV GOBIN=/web-app/go/bin

# Install database clients
RUN curl -sS https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor | tee /etc/apt/keyrings/pgdg.gpg >/dev/null \
    && echo "deb [signed-by=/etc/apt/keyrings/pgdg.gpg] http://apt.postgresql.org/pub/repos/apt noble-pgdg main" > /etc/apt/sources.list.d/pgdg.list \
    && apt-get update \
    # && apt-get install -y $MYSQL_CLIENT \
    && apt-get install -y postgresql-client-$POSTGRES_VERSION

# Clean up
RUN apt-get -y autoremove \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Create app user
RUN userdel -r ubuntu
RUN groupadd --force -g $WWWGROUP app
RUN useradd -ms /bin/bash --no-user-group -g $WWWGROUP -u 1337 -G sudo app

# Copy files
COPY start-container.sh /usr/local/bin/start-container.sh
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY . .

# Build the Go app
RUN go mod tidy
RUN mkdir -p /var/tmp/go-cache
RUN mkdir -p /web-app/go/pkg/mod
RUN mkdir -p /opt/go-bins
RUN go build -o /opt/go-bins/main ./main.go

# Set permissions
RUN chmod +x /usr/local/bin/start-container.sh
RUN chown -R app:app /var/tmp/go-cache
RUN chown -R app:app /opt/go-bins
RUN chown -R app:app /web-app

EXPOSE 8000/tcp

ENTRYPOINT ["start-container.sh"]