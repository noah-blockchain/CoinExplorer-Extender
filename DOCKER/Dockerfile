# docker build -t noah-extender:latest -f DOCKER/Dockerfile .
# docker build --no-cache -t node:latest -f DOCKER/Dockerfile .
# docker run -d -p 127.0.0.1:9000:9000 --restart=always noah-extender:latest

FROM golang:1.12-buster as builder

ENV APP_PATH /home/coin_extender

COPY . ${APP_PATH}

WORKDIR ${APP_PATH}

RUN make create_vendor && \
    make build && \
    cp ./builds/coin_extender /usr/local/bin/coin_extender

EXPOSE 10000
CMD ["coin_extender"]
STOPSIGNAL SIGTERM
