# daijinCAD-server
more docs TODO
## Build Instructions
Prerequisites:
- `go`
- `sqlc`
`sqlc` is used to generate type-safe interfaces for queries defined in `query.sql`.
```bat
:: generate database functions using sqlc
sqlc generate

:: build the executable
go build

:: run the executable
daijincad-server.exe
```
## License
AGPL