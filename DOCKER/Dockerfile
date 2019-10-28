# docker build -t noah-extender:latest -f DOCKER/Dockerfile .
# docker build --no-cache -t node:latest -f DOCKER/Dockerfile .
# docker run -d -p 127.0.0.1:9000:9000 --restart=always noah-extender:latest

FROM golang:1.12-buster as builder

ENV APP_PATH /home/noah-extender

COPY . ${APP_PATH}

WORKDIR ${APP_PATH}

RUN make create_vendor && \
    make build && \
    cp ./builds/extender /usr/local/bin/extender

EXPOSE 9000
CMD ["extender"]
STOPSIGNAL SIGTERM
