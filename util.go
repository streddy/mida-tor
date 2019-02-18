package main

import (
	"errors"
	"github.com/teamnsrg/chromedp/runner"
	"math/rand"
	"strings"
	"time"
)

// Generates the random strings which are used as identifiers for each task
// They need to be large enough to make collisions of tasks not a concern
// Currently the key space is 7.95 * 10^24
func GenRandomIdentifier() string {
	// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
	b := ""
	rand.Seed(time.Now().UTC().UnixNano())
	for i := 0; i < DefaultIdentifierLength; i++ {
		b = b + string(AlphaNumChars[rand.Intn(len(AlphaNumChars))])
	}
	return b
}

// Takes a variety of possible flag formats and puts them
// in a format that chromedp understands (key/value)
func FormatFlag(f string) (runner.CommandLineOption, error) {
	if strings.HasPrefix(f, "--") {
		f = f[2:]
	}

	parts := strings.Split(f, "=")
	if len(parts) == 1 {
		return runner.Flag(parts[0], true), nil
	} else if len(parts) == 2 {
		return runner.Flag(parts[0], parts[1]), nil
	} else {
		return runner.Flag("", ""), errors.New("Invalid flag: " + f)
	}

}

// Check to see if a flag has been removed by the RemoveBrowserFlags setting
func IsRemoved(toRemove []string, candidate string) bool {
	for _, x := range toRemove {
		if candidate == x {
			return true
		}
	}

	return false
}
