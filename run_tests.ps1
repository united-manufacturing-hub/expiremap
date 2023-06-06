# Generate coverage report
go test -v -coverprofile coverage.out ./...

# Generate html report
go tool cover -html coverage.out -o coverage.html
