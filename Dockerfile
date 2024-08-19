FROM golang:1.23.0-alpine3.19@sha256:fe8f9c7d418d3ac91787f11c31071c4814b6da5f9aae55bc581a7aacc264c395 as prepare

RUN apk update && apk add github-cli

WORKDIR /home/build
COPY . ./

RUN go build

FROM scratch

COPY --from=prepare /home/build/albedo /usr/bin/albedo

ENTRYPOINT ["/usr/bin/albedo"]
