# gofc
A go package for FC utility to manage fiber channel disk. <br>
It provides the following functions,
- Rescan host
- Get disk
- Remove disk

## Testing
Run the following integration test to check if the FC HBA card is exist,
```
go test -v -run=TestGetTargetPorts
```

Run integration test with log level
```
export GOISCSI_LOG_LEVEL=4
go test -v -run=TestRescanHost
go test -v -run=TestGetDevicesByTnameLun
go test -v -run=TestRemoveDisk
```
