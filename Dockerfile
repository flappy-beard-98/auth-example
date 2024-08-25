FROM golang:1.22-alpine AS builder
ARG SERVICE_DIR
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY ${SERVICE_DIR}/ .
COPY core/ core/
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

FROM scratch AS runner
COPY --from=builder /build/app .
EXPOSE 8080
CMD ["/app"]