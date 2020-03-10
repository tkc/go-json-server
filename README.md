
![Go](https://github.com/tkc/go-json-server/workflows/Go/badge.svg?branch=master)

```
  ___          _                 ___                      
 / __|___   _ | |__ _ ___ _ _   / __| ___ _ ___ _____ _ _ 
| (_ / _ \ | || / _` / _ \ ' \  \__ \/ -_) '_\ V / -_) '_|
 \___\___/  \__/\__,_\___/_||_| |___/\___|_|  \_/\___|_|  
                                                          
```                                                

simple and quick golang JSON mock server.  
simulate an http server and return the specified json according to a custom route.


- static json api server
- static file server
- [ ] cache response and reload
- [ ] change api.json path
- [ ] url param
- [ ] server deamon
- [ ] jwt or session auth
- [ ] error response
- [ ] access log
- [ ] E2E test sample source in Github Actions

## Install

```bash
$ go install github.com/tkc/go-json-server
```

## Prepare api.json

```bash
- api.json // required 
- response.json
```

See example  
https://github.com/tkc/go-json-server/tree/master/example

## Serve Mock Server
```bash
$ go-json-server
```

## Json API Setting

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


