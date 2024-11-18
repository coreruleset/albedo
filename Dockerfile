FROM golang:1.23.3-alpine3.19@sha256:f72297ec1cf35152ecfe7a4d692825fc608fea8f3d3fa8f986fda70184082823 AS prepare

RUN apk update && apk add github-cli

WORKDIR /home/build
COPY . ./

RUN go build

FROM scratch

COPY --from=prepare /home/build/albedo /usr/bin/albedo

ENTRYPOINT ["/usr/bin/albedo"]
