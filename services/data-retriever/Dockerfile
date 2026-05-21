FROM golang:1.26.2-alpine AS build
WORKDIR /src

COPY services/data-retriever/go.mod services/data-retriever/go.sum ./services/data-retriever/
WORKDIR /src/services/data-retriever
RUN go mod download

COPY services/data-retriever/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/data-retriever .

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=build /out/data-retriever /data-retriever
COPY --from=build /src/services/data-retriever/migrations /migrations

EXPOSE 8100
ENTRYPOINT ["/data-retriever"]
