FROM gcr.io/distroless/base
ENV BIND_ADDRESS=0.0.0.0:8080
VOLUME /storage

COPY composer-registry /usr/local/bin/composer-registry
CMD ["/usr/local/bin/composer-registry"]
