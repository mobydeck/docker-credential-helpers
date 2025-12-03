package main

import (
	"fmt"
	"io"
	"os"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/plain"
)

func main() {
	helper, err := plain.New("", "")
	if err != nil {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(1)
	}

	serve(helper)
}

func serve(helper credentials.Helper) {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stdout, usage())
		os.Exit(1)
	}

	action := os.Args[1]

	switch action {
	case "--version", "-v":
		_ = credentials.PrintVersion(os.Stdout)
		os.Exit(0)
	case "--help", "-h":
		fmt.Fprintln(os.Stdout, usage())
		os.Exit(0)
	}

	var err error
	if action == credentials.ActionStore {
		err = store(helper, os.Stdin, os.Stdout)
	} else {
		err = credentials.HandleCommand(helper, action, os.Stdin, os.Stdout)
	}

	if err != nil {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(1)
	}
}

func store(helper credentials.Helper, in io.Reader, out io.Writer) error {
	if isTerminal(in) {
		creds, err := plain.PromptForCredentials(in, out)
		if err != nil {
			return err
		}
		return helper.Add(creds)
	}

	payload, err := io.ReadAll(in)
	if err != nil {
		return err
	}

	creds, err := plain.ParseCredentialsPayload(payload)
	if err != nil {
		return err
	}

	return helper.Add(creds)
}

func isTerminal(r io.Reader) bool {
	file, ok := r.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

func usage() string {
	return fmt.Sprintf("Usage: %s <store|get|erase|list|version>", credentials.Name)
}
