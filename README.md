# Astra Commander

`Astra Commander` is a terminal dashboard for monitoring and managing Cesbo Astra instances.

![AstraCMD screenshot](https://raw.githubusercontent.com/unidiag/astracmd/refs/heads/main/dash.jpg)

The application works directly with the Astra HTTP API and WebSocket API.
It does not require a database.

## Features

- Multiple Astra connections from an INI config file
- Live terminal dashboard
- Astra online/offline status
- Astra version display
- DVB adapters list
- Streams list
- Live stream bitrate from Astra WebSocket events
- Adapter signal, quality, BER, UNC and bitrate display
- Astra log viewer
- Log filtering by selected adapter or stream
- Restart Astra
- Restart selected adapter
- Restart selected stream
- Delete selected adapter
- Delete selected stream
- Enable or disable debug log
- Set Astra license
- Access restriction by user (ROOT - full access, ANOTHER - only view and restart)
- Convenient reader management
- and etc..

## Compile
To compile you will need: `Golang 1.25.0`
```bash
sudo rm -rf /usr/local/go
wget -q https://go.dev/dl/go1.25.0.linux-amd64.tar.gz -O /tmp/go1.25.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf /tmp/go1.25.0.linux-amd64.tar.gz
/usr/local/go/bin/go version
```

After...
```bash
git clone https://github.com/unidiag/astracmd
cd ./astracmd
go build -o astracmd
```

## Run
```bash
./astracmd
./astracmd /path/to/config/astracmd.ini
```
