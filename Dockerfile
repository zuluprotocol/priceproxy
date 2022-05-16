# FILE IS AUTOMATICALLY MANAGED BY github.com/vegaprotocol/terraform//github
FROM alpine:3.15
# USER nonroot:nonroot
# COPY --chown=nonroot:nonroot bin/priceproxy /priceproxy
COPY bin/priceproxy /priceproxy
