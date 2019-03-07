FROM scratch
ADD mapping-sku-service /
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s CMD ["/mapping-sku-service","-isHealthy"]

ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT

ENTRYPOINT ["/mapping-sku-service"]