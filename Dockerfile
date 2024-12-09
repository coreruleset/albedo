FROM golang:1.23.4-alpine3.19@sha256:5f3336882ad15d10ac1b59fbaba7cb84c35d4623774198b36ae60edeba45fd84 AS prepare

RUN apk update && apk add github-cli

WORKDIR /home/build
COPY . ./

RUN go build

FROM scratch

COPY --from=prepare /home/build/albedo /usr/bin/albedo

ENTRYPOINT ["/usr/bin/albedo"]
