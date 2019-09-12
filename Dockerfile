FROM alpine:3.7 as builder

RUN echo http://nl.alpinelinux.org/alpine/v3.7/main > /etc/apk/repositories; \
    echo http://nl.alpinelinux.org/alpine/v3.7/community >> /etc/apk/repositories
    
RUN apk --no-cache add zeromq util-linux curl

RUN mkdir -p /rootfs/curl

RUN for filename in $( \
        ldd /usr/bin/curl \
        # extract unqiue names
        | grep -oE "/[^:]*" | awk '{print $1}' | sort -u); \        
    do \
      cp -duv $filename `realpath $filename` --parents /rootfs/curl/; \
    done


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

# CURL libraries
COPY --from=builder /usr/bin/curl /usr/bin/
COPY --from=builder /rootfs/curl /

ADD product-data-service /
ADD res/docker/ /res

ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT

CMD ["/product-data-service","-r","--profile=docker","--confdir=/res"]