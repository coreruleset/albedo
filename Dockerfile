FROM alpine:latest as prepare

RUN apk update && apk add github-cli

RUN gh release download -R coreruleset/albedo

FROM scratch

COPY --from=prepare albedo /usr/bin/albedo

ENTRYPOINT ["/usr/bin/albedo"]
