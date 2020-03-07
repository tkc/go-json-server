
![Go](https://github.com/tkc/go-json-server/workflows/Go/badge.svg?branch=master)

```
/ __|___   _ | |__ _ ___ _ _   / __| ___ _ ___ _____ 
| (_ / _ \ | || / _` / _ \ ' \  \__ \/ -_) '_\ V / -_)
 \___\___/  \__/\__,_\___/_||_| |___/\___|_|  \_/\___|
```                                                

simple and quick golang JSON mock server

- static json api server
- static file server

- [ ] cache response and reload
- [ ] change api.json path
- [ ] url param
- [ ] server deamon
- [ ] jwt or session auth
- [ ] error response
- [ ] access log

## Install

```bash
go install github.com/tkc/go-json-server
```

## Serve Mock Server
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
      "path": "/",
      "jsonPath": "./health-check.json"
    },
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


`health-check.json`
```javascript
{
    "status": "ok",
    "message": "go-json-server started"
}
```

`users.json`
```javascript
[
  {
    "id":1,
    "name": "name"
  }
]
```

## License

MIT ✨


