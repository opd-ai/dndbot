FROM alpine:edge
RUN apk update
RUN apk upgrade
RUN apk add go make git
RUN addgroup -S user && adduser -S user -G user -h /home/user
USER user
WORKDIR /home/user
RUN go install github.com/opd-ai/dndbot/srv@latest
CMD ~/go/bin/srv -paywall=true -tls=true