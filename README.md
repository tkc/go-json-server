# go-json-server

simple and quick golang JSON mock server

## Install

```bash
go install testpkg/tkc/go-json-server
```

## Run
```bash
go-json-server
```

See example  
https://github.com/tkc/go-json-server/tree/master/example

## API Setting
put api.json  and run `go-json-server`

`api.json`

```javascript
{
  "port": 3000,
  "endpoints": [
    {
      "method": "GET",
      "status": 200,
      "path": "/users",
      "jsonPath": "./users.json"
    },
    {
      "method": "GET",
      "status": 200,
      "path": "/user/1",
      "jsonPath": "./user.json"
    },
    {
      "path": "/file",
      "folder": "./static"
    }
  ]
}
```

`user.json`
```javascript
{
  "id": 1,
  "name": "name",
  "address": "address"
}
```
## License

MIT ✨


