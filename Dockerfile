FROM gcr.io/distroless/base
COPY composer-registry /usr/local/bin/composer-registry
CMD ["/usr/local/bin/composer-registry"]
