FROM gcr.io/distroless/static-debian12:nonroot
COPY pidgr-mcp /pidgr-mcp
ENTRYPOINT ["/pidgr-mcp"]
