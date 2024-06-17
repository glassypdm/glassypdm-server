# daijinCAD-server
## TODO
- [ ] file upload routes `/ingest` `/ingestreq`
- [ ] POST `/commit` route with db transaction
- [ ] route where we can get new files since a commit ID for a project
- [ ] `/project/info` return if user can manage project permissions/settings

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