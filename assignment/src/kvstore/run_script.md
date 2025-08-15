```$env:PORT = "8001"; `
$env:SELF_URL = "http://localhost:8001"; `
$env:PEERS = "http://localhost:8001,http://localhost:8002"; `
go run main.go```

```$env:PORT = "8002"; `
$env:SELF_URL = "http://localhost:8002"; `
$env:PEERS = "http://localhost:8001"; `
go run main.go```

```$env:PORT = "8003"; `
$env:SELF_URL = "http://localhost:8003"; `
$env:PEERS = "http://localhost:8001"; `
go run main.go```

```$env:PORT = "8004"; `
$env:SELF_URL = "http://localhost:8004"; `
$env:PEERS = "http://localhost:8001"; `
go run main.go```