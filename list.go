package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func listInfoRefs() error {
	refsCat, err := ipfsShell.Cat(filepath.Join(ipfsRepoPath, "info", "refs"))
	if err != nil {
		return fmt.Errorf("failed to cat info/refs from %s: %w", ipfsRepoPath, err)
	}
	s := bufio.NewScanner(refsCat)
	for s.Scan() {
		hashRef := strings.Split(s.Text(), "\t")
		if len(hashRef) != 2 {
			return fmt.Errorf("processing info/refs: what is this: %v", hashRef)
		}
		ref2hash[hashRef[1]] = hashRef[0]
	}
	if err := s.Err(); err != nil {
		return fmt.Errorf("ipfs.Cat(info/refs) scanner error: %w", err)
	}
	return nil
}

func listHeadRef() (string, error) {
	headCat, err := ipfsShell.Cat(filepath.Join(ipfsRepoPath, "HEAD"))
	if err != nil {
		return "", fmt.Errorf("failed to cat HEAD from %s: %w", ipfsRepoPath, err)
	}
	head, err := ioutil.ReadAll(headCat)
	if err != nil {
		return "", fmt.Errorf("failed to readAll HEAD from %s: %w", ipfsRepoPath, err)
	}
	if !bytes.HasPrefix(head, []byte("ref: ")) {
		return "", fmt.Errorf("illegal HEAD file from %s: %q", ipfsRepoPath, head)
	}
	headRef := string(bytes.TrimSpace(head[5:]))
	headHash, ok := ref2hash[headRef]
	if !ok {
		// use first hash in map?..
		return "", fmt.Errorf("unknown HEAD reference %q", headRef)
	}
	return headHash, headCat.Close()
}

func listIterateRefs(forPush bool) error {
	refsDir := filepath.Join(ipfsRepoPath, "refs")
	return Walk(refsDir, func(p string, info *shell.LsLink, err error) error {
		if err != nil {
			return fmt.Errorf("walk(%s) failed: %w", p, err)
		}
		log.Log("event", "debug", "name", info.Name, "msg", "iterateRefs: walked to", "p", p)
		if info.Type == 2 {
			rc, err := ipfsShell.Cat(p)
			if err != nil {
				return fmt.Errorf("walk(%s) cat ref failed: %w", p, err)
			}
			data, err := ioutil.ReadAll(rc)
			if err != nil {
				return fmt.Errorf("walk(%s) readAll failed: %w", p, err)
			}
			if err := rc.Close(); err != nil {
				return fmt.Errorf("walk(%s) cat close failed: %w", p, err)
			}
			sha1 := strings.TrimSpace(string(data))
			refName := strings.TrimPrefix(p, ipfsRepoPath+"/")
			ref2hash[refName] = sha1
			log.Log("event", "debug", "refMap", ref2hash, "msg", "ref2hash map updated")
		}
		return nil
	})
}

// semi-todo make shell implement http.FileSystem
// then we can reuse filepath.Walk and make a lot of other stuff simpler
var SkipDir = fmt.Errorf("walk: skipping")

type WalkFunc func(path string, info *shell.LsLink, err error) error

func walk(path string, info *shell.LsLink, walkFn WalkFunc) error {
	err := walkFn(path, info, nil)
	if err != nil {
		if info.Type == 1 && err == SkipDir {
			return nil
		}
		return err
	}
	if info.Type != 1 {
		return nil
	}
	list, err := ipfsShell.List(path)
	if err != nil {
		log.Log("msg", "walk list failed", "err", err)

		return walkFn(path, info, err)
	}
	for _, lnk := range list {
		fname := filepath.Join(path, lnk.Name)
		err = walk(fname, lnk, walkFn)
		if err != nil {
			if lnk.Type != 1 || err != SkipDir {
				return err
			}
		}
	}
	return nil
}

func Walk(root string, walkFn WalkFunc) error {
	list, err := ipfsShell.List(root)
	if err != nil {
		log.Log("msg", "walk root failed", "err", err)
		return walkFn(root, nil, err)
	}
	for _, l := range list {
		fname := filepath.Join(root, l.Name)
		if err := walk(fname, l, walkFn); err != nil {
			return err
		}
	}
	return nil
}
