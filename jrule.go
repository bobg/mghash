package mghash

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"sort"

	json "github.com/gibson042/canonicaljson-go"
	"github.com/magefile/mage/mg"
	"github.com/pkg/errors"
)

// JRule is a Rule that lists a set of source files and a set of target files,
// and includes a command for producing targets from sources.
type JRule struct {
	Sources []string `json:"sources"`
	Targets []string `json:"targets"`
	Command []string `json:"command"`
}

var _ Rule = JRule{}

func (jr JRule) RuleHash() []byte {
	jr2 := JRule{
		Sources: make([]string, len(jr.Sources)),
		Targets: make([]string, len(jr.Targets)),
		Command: jr.Command,
	}
	copy(jr2.Sources, jr.Sources)
	copy(jr2.Targets, jr.Targets)
	sort.Strings(jr2.Sources)
	sort.Strings(jr2.Targets)
	j, _ := json.Marshal(jr2)
	sum := sha256.Sum256(j)
	return sum[:]
}

func (jr JRule) ContentHash(_ context.Context) ([]byte, error) {
	s := struct {
		Sources map[string][]byte `json:"sources"`
		Targets map[string][]byte `json:"targets"`
		Command []string          `json:"command"`
	}{
		Sources: make(map[string][]byte),
		Targets: make(map[string][]byte),
		Command: jr.Command,
	}
	err := fillWithFileHashes(jr.Sources, s.Sources)
	if err != nil {
		return nil, errors.Wrap(err, "computing source hash(es)")
	}
	err = fillWithFileHashes(jr.Targets, s.Targets)
	if err != nil {
		return nil, errors.Wrap(err, "computing target hash(es)")
	}
	j, err := json.Marshal(s)
	if err != nil {
		return nil, errors.Wrap(err, "in JSON marshaling")
	}
	sum := sha256.Sum256(j)
	fmt.Printf("xxx jr.Hash for %v: %x\n", jr.Targets, sum[:])
	return sum[:], nil
}

func (jr JRule) Run(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, jr.Command[0], jr.Command[1:]...)
	if mg.Verbose() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

func fillWithFileHashes(files []string, hashes map[string][]byte) error {
	for _, file := range files {
		h, err := hashFile(file)
		if errors.Is(err, fs.ErrNotExist) {
			h = nil
		} else if err != nil {
			return errors.Wrapf(err, "computing hash of %s", file)
		}
		hashes[file] = h
	}
	return nil
}

func hashFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "opening %s", path)
	}
	defer f.Close()
	hasher := sha256.New()
	_, err = io.Copy(hasher, f)
	if err != nil {
		return nil, errors.Wrapf(err, "hashing %s", path)
	}
	return hasher.Sum(nil), nil
}