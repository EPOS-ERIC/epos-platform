FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY services/data-retriever/data-retriever /data-retriever
COPY services/data-retriever/migrations /migrations

EXPOSE 8100
ENTRYPOINT ["/data-retriever"]
