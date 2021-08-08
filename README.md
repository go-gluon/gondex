# gondex
GO source code indexer

Create indexer
```go
indexer := gondex.CreateIndexer()
if e := indexer.Load(); e != nil {
    panic(e)
}
```

Find all structs by annotation
```go
items := indexer.FindStructByAnnotation("gluon:Config")
for _, item := range items {
    fmt.Printf("Struct: %v\n", item.Id())
}
```

Find all implementation of the interface
```go
s := indexer.FindInterfaceImplementation("github.com/go-gluon/generator/test/user.TestI")
for _, t := range s {
    fmt.Printf("Struct: %v\n", t.Id())
}    
```
