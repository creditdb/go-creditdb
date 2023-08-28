# CreditDB Client for Go

This package provides a client library for CreditDB, a fast and efficient key-value database built in Typescript.

## Installation

```sh
go get github.com/creditdb/go-creditdb
```

## Usage

Import the package into your code:

```go
import "github.com/creditdb/go-creditdb"
```
```go
func main(){
    client := creditdb.NewClient().WithPage(1)
    ctx:=context.Background()
	defer client.Close(ctx)
	x, err:=client.Ping(ctx)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Println(x)
}
```
## License

[MIT](LICENSE)