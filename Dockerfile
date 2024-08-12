FROM golang:1.22.6-alpine3.19@sha256:1bad39361dd21f2f881ce10ff810e40e5be3eba89a0b61e762e05ec42f9bbaf2 as prepare

RUN apk update && apk add github-cli

WORKDIR /home/build
COPY . ./

RUN go build

FROM scratch

COPY --from=prepare /home/build/albedo /usr/bin/albedo

ENTRYPOINT ["/usr/bin/albedo"]
