# go-json-server

simple add quick golang json server

# Install

```$xslt
go install testpkg/tkc/go-json-server
```

## Run
```$xslt
go-json-server
```

See example  
https://github.com/tkc/go-json-server/tree/master/example

## API setting
put api.json  and run `go-json-server`

```
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