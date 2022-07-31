package mghash

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

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
	Dir     string   `json:"dir"`
}

var _ Rule = JRule{}

func (jr JRule) String() string {
	return fmt.Sprintf("JRule[%s]", strings.Join(jr.Targets, " "))
}

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
	// Theory of operation:
	// A new struct is built out of the fields of jr,
	// but with Sources and Targets mapped to each file's hash,
	// where the file exists
	// (and nil where it doesn't).
	// The struct is JSON marshaled
	// (using canonical-json for reproducibility)
	// and hashed.
	// Any change to the set of sources or targets,
	// the presence of absence of any file,
	// the content of any file,
	// or the strings in jr.Command
	// will change the hash.

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
	return sum[:], nil
}

func (jr JRule) Run(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, jr.Command[0], jr.Command[1:]...)
	cmd.Dir = jr.Dir
	if mg.Verbose() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		log.Printf("Running %s %s", jr.Command[0], strings.Join(jr.Command[1:], " "))
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

// JDir parses a file named .mghash.json in the given directory,
// if there is one,
// returning the JRules it contains.
// The default directory for any JRules not specifying one is dir.
func JDir(dir string) ([]JRule, error) {
	f, err := os.Open(filepath.Join(dir, ".mghash.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "opening %s/.msghash.json")
	}
	defer f.Close()
	var (
		result []JRule
		dec    = json.NewDecoder(f)
	)
	for dec.More() {
		var j JRule
		if err = dec.Decode(&j); err != nil {
			return errors.Wrapf(err, "parsing %s/.mghash.json")
		}
		if j.Dir == "" {
			j.Dir = dir
		}
		result = append(result, j)
	}
	return result, nil
}

// JTree walks the tree rooted at dir,
// looking for .mghash.json files
// and parsing the JRules out of them using JDir.
func JTree(dir string) ([]JRule, error) {
	var result []JRule
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		j, err := JDir(path)
		if err != nil {
			return err
		}
		result = append(result, j...)
		return nil
	})
	return result, err
}
