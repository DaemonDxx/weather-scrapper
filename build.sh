set -e
export GOFLAGS="-mod=vendor"

go build -o bin/ cmd/parser/parser.go
go build -o bin/ cmd/scrapper/scrapper.go