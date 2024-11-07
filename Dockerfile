FROM golang:1.23.3-alpine3.19@sha256:36cc30986d1f9bc46670526fe6553b078097e562e196344dea6a075e434f8341 AS prepare

RUN apk update && apk add github-cli

WORKDIR /home/build
COPY . ./

RUN go build

FROM scratch

COPY --from=prepare /home/build/albedo /usr/bin/albedo

ENTRYPOINT ["/usr/bin/albedo"]
