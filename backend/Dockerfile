FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY gateway-linux /gateway
EXPOSE 8080
CMD ["/gateway"]
