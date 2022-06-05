# Mghash

This is mghash, an extension to [Mage](https://magefile.org/) for computing hash-based dependencies.

Mage already includes functions for skipping recompilation of targets whose file modtimes are newer than those of their sources.
This library does the same thing, but based on content hashes, not file modtimes.

# Example

This Magefile snippet defines a rule called Generate
for generating `foo.pb.go` from `foo.proto`.
Before running `protoc`,
the contents of `foo.proto` and `foo.pb.go` are hashed together with the text of the `protoc` command.
If the resulting hash is present in the `hashes.sqlite` databaase
then `foo.pb.go` is considered to be up to date and is not rebuilt.
Otherwise the `protoc` command runs,
after which the hash is recomputed and added to the database.

```go
import (
  "github.com/bobg/mghash"
  "github.com/bobg/mghash/sqlite"
)

func Generate() error {
  db, err := sqlite.Open("hashes.sqlite")
  if err != nil { ... }
  defer db.Close()
  
  mg.Deps(&mghash.Fn{DB: db, Rule: mghash.JRule{
    Sources: []string{"foo.proto"},
    Targets: []string{"foo.pb.go"},
    Command: []string{"protoc", "-I.", "--go_out=.", "foo.proto"},
  }})
}
```
