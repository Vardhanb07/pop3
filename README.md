# pop3

A minimal POP3 server implementation based on RFC 1939.

## Setup

1. Ensure Go is installed.
2. Start the server:

```
go run .
```

The server listens on port 5000 by default.

## Test

Open another terminal and connect using netcat:

```
nc -C localhost 5000
```

You should see the POP3 greeting. Then try simple commands:

```
USER test@test.com
PASS test
STAT
LIST
QUIT
```

If you want a different port, update the listener in main.go.
