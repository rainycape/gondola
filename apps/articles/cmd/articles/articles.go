// The articles command is used for easily managing article files.
//
// To install this tool, type go install gnd.la/apps/articles/cmd/articles.
//
// Then, view its help and usage by typing:
//
//  articles help
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gnd.la/apps/articles/article"

	"gopkgs.com/command.v1"
)

func openArticle(p string) (*article.Article, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("error opening %s: %s", p, err)
	}
	defer f.Close()
	article, err := article.New(f)
	if err != nil {
		return nil, fmt.Errorf("error loading %s: %s", p, err)
	}
	return article, nil
}

func dumpCommand(args []string) {
	for _, v := range args {
		article, err := openArticle(v)
		if err != nil {
			log.Print(err)
			continue
		}
		fmt.Printf("article %s\n%+v\n", v, article)
	}
}

func setCommand(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("invalid number of arguments %d, must be <property> <value> <article>", len(args))
	}
	p := args[2]
	article, err := openArticle(p)
	if err != nil {
		return err
	}
	if err := article.Set(args[0], args[1]); err != nil {
		return err
	}
	var buf bytes.Buffer
	if _, err := article.WriteTo(&buf); err != nil {
		return err
	}
	return ioutil.WriteFile(p, buf.Bytes(), 0644)
}

const (
	dumpHelp = `The dump command dumps the articles it parses to the
standard output. It can be used to validate articles files.`
	setHelp = `The set command sets properties in article files. The parsed
properties include:

    - id (string)
    - title ([]string)
    - slug ([]string)
    - synopsis (string)
    - updated ([]time.Time)
    - priority (int)

Setting a property stored in a slice prepends the new value to the existing ones. To
delete a value, simply remove its line with a text editor.

time.Time properties must use one of the following formats:

    - now
    - today
    - yyyy-mm-dd
    - yyyy-mm-dd hh:MM
    - yyyy-mm-dd hh:MM:ss
    - time.RFC822
    - time.RFC822Z`
)

var (
	commands = []*command.Cmd{
		{
			Name:     "dump",
			Help:     "Dump parsed articles to command line",
			LongHelp: dumpHelp,
			Usage:    "<article-1> [article-2] ... [article-n]",
			Func:     dumpCommand,
		},
		{
			Name:     "set",
			Help:     "Set or add a property to the given article",
			LongHelp: setHelp,
			Usage:    "<property> <value> <article>",
			Func:     setCommand,
		},
	}
)

func main() {
	command.Run(commands)
}
