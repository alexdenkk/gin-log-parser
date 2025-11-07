# gin-log-parser
Cli parser for gin logs \
Can output metrics or raw/json logs by filter

# Requiremetns
 - **Go 1.24+**

# Install

```sh
go build -o ginlog cmd/parser/main.go
mv ginlog /usr/bin
```

# Usage
**\*works with debug mode too**

Help:
```sh
ginlog -help
```
Example usage:
```
cat log.txt | ginlog -method GET
```
