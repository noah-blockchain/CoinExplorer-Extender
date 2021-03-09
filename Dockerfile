# docker build -t noah-extender:latest -f Dockerfile .
# docker build --no-cache -t noah-extender:latest -f Dockerfile .
# docker run -d -p 127.0.0.1:10000:10000 --restart=always noah-extender:latest

FROM golang:1.12-buster as builder
ENV APP_PATH /home/coin_extender
COPY . ${APP_PATH}
WORKDIR ${APP_PATH}
RUN make build

FROM debian:buster-slim as executor
COPY --from=builder /home/coin_extender/build/coin_extender /usr/local/bin/coin_extender
COPY --from=builder /home/coin_extender/migrations /migrations
CMD ["coin_extender"]
STOPSIGNAL SIGTERM
