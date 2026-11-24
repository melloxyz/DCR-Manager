@echo off
pushd "%~dp0"
if not exist bin mkdir bin
go test ./...
if errorlevel 1 goto fail
go build -trimpath -ldflags "-s -w" -o bin/dcr-manager.exe ./cmd/dcr-manager
if errorlevel 1 goto fail
popd
exit /b 0
:fail
popd
exit /b 1
