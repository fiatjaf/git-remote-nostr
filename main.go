package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
)

var (
	ref2hash = make(map[string]string)

	repo   string // local repository dir
	remote string // nostr id
)
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})

func main() {
	repo = os.Getenv("GIT_DIR")
	if repo == "" {
		log.Fatal().Msg("could not get GIT_DIR env var")
		return
	}
	if repo == ".git" {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal().Err(err).Str("repo", repo).Msg("failed to get current directory")
			return
		}
		repo = filepath.Join(cwd, ".git")
	}

	if len(os.Args[1:]) != 2 {
		log.Fatal().Msg("must be called with 2 args only (this is meant to be called by git internally, do not call it manually)")
		return
	}

	repoURL := os.Args[2]
	remote = strings.Split(repoURL, "://")[1]

	// answer git commands
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		fmt.Fprintf(os.Stderr, "got line: %s\n", text)
		switch text {
		case "capabilities":
			fmt.Println("fetch")
			fmt.Println("push")
			fmt.Println("")
		case "list", "list for-push":
			var (
				forPush = strings.Contains(text, "for-push")
				err     error
				head    string
			)
			if err = listInfoRefs(); err == nil { // try .git/info/refs first
				if head, err = listHeadRef(); err != nil {
					log.Fatal().Err(err).Msg("failed to list head")
					return
				}
			} else { // alternativly iterate over the refs directory like git-remote-dropbox
				if forPush {
					log.Debug().Msg("for-push: should be able to push to non existant.. TODO #2")
				} else {
					log.Debug().Err(err).Msg("didn't find info/refs in repo, falling back...")
					if err = listIterateRefs(forPush); err != nil {
						log.Fatal().Err(err).Msg("failed to iterate over refs")
						return
					}
				}
			}
			if len(ref2hash) == 0 {
				log.Fatal().Msg("did not find _any_ refs")
				return

			}
			// output
			for ref, hash := range ref2hash {
				if head == "" && strings.HasSuffix(ref, "master") {
					// guessing head if it isnt set
					head = hash
				}
				fmt.Printf("%s %s\n", hash, ref)
			}
			fmt.Printf("%s HEAD\n", head)
			fmt.Println("")
		}
		// case "fetch":
		// 	for scanner.Scan() {
		// 		spl := strings.Split(text, " ")
		// 		if len(spl) < 2 {
		// 			log.Fatal().Str("cmd", text).Msg("malformed 'fetch' command")
		// 			return
		// 		}
		// 		err := fetchObject(spl[1])
		// 		if err == nil {
		// 			fmt.Println("")
		// 			continue
		// 		}

		// 		// TODO isNotExist(err) would be nice here
		// 		// log.Log("sha1", fetchSplit[1], "name", fetchSplit[2], "err", err, "msg", "fetchLooseObject failed, trying packed...")

		// 		err = fetchPackedObject(spl[1])
		// 		if err != nil {
		// 			return
		// 		}
		// 		text = scanner.Text()
		// 		if text == "" {
		// 			break
		// 		}
		// 	}
		// 	fmt.Println("")
		// case "push":
		// 	for scanner.Scan() {
		// 		pushSplit := strings.Split(text, " ")
		// 		if len(pushSplit) < 2 {
		// 			return errors.Errorf("malformed 'push' command. %q", text)
		// 		}
		// 		srcDstSplit := strings.Split(pushSplit[1], ":")
		// 		if len(srcDstSplit) < 2 {
		// 			return errors.Errorf("malformed 'push' command. %q", text)
		// 		}
		// 		src, dst := srcDstSplit[0], srcDstSplit[1]
		// 		f := []interface{}{
		// 			"src", src,
		// 			"dst", dst,
		// 		}
		// 		log.Log(append(f, "msg", "got push"))
		// 		if src == "" {
		// 			fmt.Printf("error %s %s\n", dst, "delete remote dst: not supported yet - please open an issue on github")
		// 		} else {
		// 			if err := push(src, dst); err != nil {
		// 				fmt.Printf("error %s %s\n", dst, err)
		// 				return err
		// 			}
		// 			fmt.Println("ok", dst)
		// 		}
		// 		text = scanner.Text()
		// 		if text == "" {
		// 			break
		// 		}
		// 	}
		// 	fmt.Println("")
		// case "":
		// 	break
		// default:
		// 	return
		// }
	}
	if err := scanner.Err(); err != nil {
		log.Fatal().Err(err).Msg("error scanning stdin")
		return
	}
}
