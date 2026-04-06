FROM golang:1.24.2 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY src ./src

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
	go build -trimpath -ldflags="-s -w" -o /out/agrafa-agent .

FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app

COPY --from=build /out/agrafa-agent /usr/local/bin/agrafa-agent
COPY health_checks.json.example /app/health_checks.json.example

ENTRYPOINT ["/usr/local/bin/agrafa-agent"]
