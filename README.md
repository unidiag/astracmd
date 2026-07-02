# Astra Commander

`Astra Commander` is a terminal dashboard for monitoring and managing Cesbo Astra instances.

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

## Compile
Need: `Golang 1.25.0` `NodeJS v.18.19.1`
```bash
cd ./web
npm update
npm run build
cd ..
go build -o astracmd
```

## Run
```bash
./astracmd
./astracmd /path/to/config/astracmd.ini
./astracmd 5000
```
