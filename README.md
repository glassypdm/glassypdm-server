# daijinCAD-server
## Build Instructions
Prerequisites:
- `go`
- `sqlc`
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
## notes
- to upload to a project: have team permission to add people/create projects OR project permission to write