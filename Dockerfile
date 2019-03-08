FROM scratch
ADD product-data-service /
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s CMD ["/product-data-service","-isHealthy"]

ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT

ENTRYPOINT ["/product-data-service"]