# daijinCAD-server
## TODO
- [x] file upload routes `/ingest` `/ingestreq`
- [x] POST `/commit` route with db transaction
- [x] `/project/info` return if user can manage project permissions/settings

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