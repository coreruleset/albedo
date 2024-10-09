FROM golang:1.23.2-alpine3.19@sha256:f6392ffebb028fed5ffe743ddb9716e38402c978779edd66474bb5d05f5e65e4 AS prepare

RUN apk update && apk add github-cli

WORKDIR /home/build
COPY . ./

RUN go build

FROM scratch

COPY --from=prepare /home/build/albedo /usr/bin/albedo

ENTRYPOINT ["/usr/bin/albedo"]
