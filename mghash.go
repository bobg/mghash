package mghash

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"reflect"
	"runtime"

	json "github.com/gibson042/canonicaljson-go"
	"github.com/magefile/mage/mg"
	"github.com/pkg/errors"
)

// Fn is an mg.Fn
// (see https://pkg.go.dev/github.com/magefile/mage/mg#Fn)
// that knows how to skip rebuilding a target
// that is up-to-date with respect to its sources.
// "Up-to-date" here does not refer to file modtimes,
// but rather to content hashes:
// the existing target was computed from sources
// that are byte-for-byte the same now
// as they were when the target was built.
type Fn struct {
	DB   DB
	Rule Rule
}

// Rule knows how to report a hash representing itself,
// and another hash representing itself plus the state of all sources and targets;
// and how to produce its targets from its sources.
type Rule interface {
	fmt.Stringer

	// RuleHash produces the hash of this rule.
	// This should be a strong, collision-resistant value
	// that is sensitive to changes in the rule itself
	// but not in any of its sources or targets.
	RuleHash() []byte

	// ContentHash produces a hash that incorporates information about the rule
	// combined with the state of all sources and targets.
	// This should be a strong, collision-resistant value.
	ContentHash(context.Context) ([]byte, error)

	// Run is a function that can generate this rule's targets.
	Run(context.Context) error
}

// DB is a database for storing hashes.
// It must permit concurrent operations safely.
// It may expire entries to save space.
type DB interface {
	// Has tells whether the database contains the given entry.
	Has(context.Context, []byte) (bool, error)

	// Add adds an entry to the database.
	Add(context.Context, []byte) error
}

var _ mg.Fn = &Fn{}

// Name implements mg.Fn.
func (f *Fn) Name() string {
	// As suggested at https://pkg.go.dev/github.com/magefile/mage@v1.13.0/mg#Fn.
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

// ID implements mg.Fn.
func (f *Fn) ID() string {
	s := struct {
		Name     string `json:"name"`
		RuleHash []byte `json:"rule_hash"`
	}{
		Name:     f.Name(),
		RuleHash: f.Rule.RuleHash(),
	}
	j, _ := json.Marshal(s)
	sum := sha256.Sum256(j)
	return hex.EncodeToString(sum[:])
}

// Run implements mg.Fn.
func (f *Fn) Run(ctx context.Context) error {
	h, err := f.Rule.ContentHash(ctx)
	if err != nil {
		return errors.Wrap(err, "computing content hash")
	}
	ok, err := f.DB.Has(ctx, h)
	if err != nil {
		return errors.Wrap(err, "consulting hash DB")
	}
	if ok {
		if mg.Verbose() {
			log.Printf("%s up to date", f.Rule)
		}
		return nil
	}
	if err = f.Rule.Run(ctx); err != nil {
		return errors.Wrap(err, "in Run")
	}
	h, err = f.Rule.ContentHash(ctx)
	if err != nil {
		return errors.Wrap(err, "recomputing content hash")
	}
	return f.DB.Add(ctx, h)
}
