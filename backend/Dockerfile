# Backend monorepo image: one image per service via --build-arg SERVICE.
# docker build --build-arg SERVICE=catalog -t vitrine/catalog:1.0 .
FROM golang:1.22-alpine AS build
ARG SERVICE
ENV CGO_ENABLED=0 \
    GOFLAGS=-mod=mod
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -trimpath -o /out/svc ./services/${SERVICE}

FROM alpine:3.20
RUN adduser -D -H app
USER app
COPY --from=build /out/svc /app/svc
# PORT is provided by the runtime (8080-8084 locally, $PORT on Cloud Run).
ENTRYPOINT ["/app/svc"]
