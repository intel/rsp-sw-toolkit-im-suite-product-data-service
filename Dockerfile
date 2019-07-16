FROM alpine:3.7 as builder

RUN echo http://nl.alpinelinux.org/alpine/v3.7/main > /etc/apk/repositories; \
    echo http://nl.alpinelinux.org/alpine/v3.7/community >> /etc/apk/repositories
    
RUN apk --no-cache add zeromq util-linux


FROM busybox:1.30.1

# ZeroMQ libraries and dependencies
COPY --from=builder /lib/libc.musl-x86_64.so.1 /lib/
COPY --from=builder /lib/ld-musl-x86_64.so.1 /lib/
COPY --from=builder /usr/lib/libzmq.so.5.1.5 /usr/lib/
COPY --from=builder /usr/lib/libzmq.so.5 /usr/lib/
COPY --from=builder /usr/lib/libsodium.so.23 /usr/lib/ 
COPY --from=builder /usr/lib/libstdc++.so.6 /usr/lib/
COPY --from=builder /usr/lib/libgcc_s.so.1 /usr/lib/
COPY --from=builder /usr/lib/libcrypto.so.42 /usr/lib/
COPY --from=builder /usr/lib/libcrypto.so.42.0.0 /usr/lib/

ADD product-data-service /
ADD res/docker/ /res
HEALTHCHECK --interval=5s --timeout=3s CMD ["/product-data-service","-isHealthy"]

ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT

CMD ["/product-data-service", "--profile=docker","--confdir=/res"]