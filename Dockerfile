FROM golang:1.16.3 as builder
LABEL maintainer="Frank Kloeker <eumel@arcor.de>"

RUN mkdir /rds

WORKDIR /rds
ADD . /rds

RUN go mod download && go mod tidy && go vet . && go build -ldflags="-s -w" -o rds rds.go
RUN rm -Rf models routers vendor \
    && rm -f Dockerfile go.mod go.sum

FROM gcr.io/distroless/static
WORKDIR /rds
COPY --from=builder /rds/rds /rds/rds
CMD ["./rds"]
