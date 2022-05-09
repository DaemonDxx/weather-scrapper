FROM golang:1.17-alpine as build
RUN apk add build-base

WORKDIR /src

COPY ./ ./

RUN GOFLAGS="-mod=vendor" go build -o ./bin/app ./cmd/scrapper/scrapper.go

FROM golang:1.17-alpine as prod

WORKDIR /app

COPY --from=build /src/bin/ ./

CMD ["./app"]