FROM alpine:edge
RUN apk update
RUN apk upgrade
RUN apk add go make git
RUN go install github.com/opd-ai/dndbot/srv@latest
CMD ~/go/bin/srv -paywall=true -tls=true