# go-reverse-proxy

To build and run the project:

```
go build
./go-reverse-proxy
```

I also included a small go program which starts two backend servers (from a corresponding config file):

```
cd backend_servers/
go build start_backend.go
./start_backend
```

Here is a list of curl commands for testing the reverse proxy:

1. Valid token request:

```
curl -H "X-Auth-Token: valid-token-1" http://localhost:8080/some/path
```

2. Invalid token request:

```
curl -H "X-Auth-Token: invalid-token" http://localhost:8080/some/path
```

3. No token:

```
curl http://localhost:8080/some/path
```

4. Websocket request with a valid token:

```
websocat -H="X-Auth-Token: valid-token-1" ws://localhost:8080/ws
```

5. Websocket request with an invalid token:

```
websocat -H="X-Auth-Token: invalid-token" ws://localhost:8080/ws
```

6. Websocket request with no token:

```
websocat ws://localhost:8080/ws
```
