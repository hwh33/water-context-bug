# water-context-bug

A program demonstrating a minor issue with refraction-networking/water. The library hangs on to the dial context and closes the connection when this context completes. The connection should instead stay open.

Demonstration:
```
➜  water-context-bug git:(main) go build -o example main.go
➜  water-context-bug git:(main) ✗ ./example -h
Usage of ./example:
  -cancel-dial-ctx
    	whether to cancel the dial context after message 1 (default true)
  -use-tcp
    	test with TCP instead of WATER
  -wasm string
    	path to the transport WASM file (default "plain.go.wasm")

# Everything works as expected over plain TCP.
➜  water-context-bug git:(main) ✗ ./example -use-tcp=true
write succeeded: message 0
read succeeded: message 0
write succeeded: message 1
read succeeded: message 1
cancelling dial
write succeeded: message 2
read succeeded: message 2

# Switching to WATER, we see that the connection closes when the dial context is cancelled.
➜  water-context-bug git:(main) ✗ ./example -use-tcp=false
2024/07/18 16:14:35 WARN water: host_defer function is imported by WATM, it is deprecated and will NOT be executed when WATM exits
2024/07/18 16:14:35 WARN water: function env.pull_config is not imported.
2024/07/18 16:14:35 WARN water: host_defer function is imported by WATM, it is deprecated and will NOT be executed when WATM exits
2024/07/18 16:14:35 WARN water: function env.pull_config is not imported.
write succeeded: message 0
2024/07/18 22:14:35 worker: working as dialer
2024/07/18 22:14:35 worker: working as listener
read succeeded: message 0
2024/07/18 16:14:35 WARN water: host_defer function is imported by WATM, it is deprecated and will NOT be executed when WATM exits
2024/07/18 16:14:35 WARN water: function env.pull_config is not imported.
write succeeded: message 1
read succeeded: message 1
cancelling dial
write succeeded: message 2
read failed: read tcp 127.0.0.1:49282->127.0.0.1:49284: read: connection reset by peer

# If we leave the dial context open, the connection stays open. However, we should not need to do this.
➜  water-context-bug git:(main) ✗ ./example -use-tcp=false -cancel-dial-ctx=false
2024/07/18 16:14:47 WARN water: host_defer function is imported by WATM, it is deprecated and will NOT be executed when WATM exits
2024/07/18 16:14:47 WARN water: function env.pull_config is not imported.
2024/07/18 16:14:47 WARN water: host_defer function is imported by WATM, it is deprecated and will NOT be executed when WATM exits
2024/07/18 16:14:47 WARN water: function env.pull_config is not imported.
write succeeded: message 0
2024/07/18 22:14:47 worker: working as dialer
2024/07/18 22:14:47 worker: working as listener
read succeeded: message 0
2024/07/18 16:14:47 WARN water: host_defer function is imported by WATM, it is deprecated and will NOT be executed when WATM exits
2024/07/18 16:14:47 WARN water: function env.pull_config is not imported.
write succeeded: message 1
read succeeded: message 1
write succeeded: message 2
read succeeded: message 2
```