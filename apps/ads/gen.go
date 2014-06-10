// +build IGNORE

package main

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gnd.la/internal/gen/genutil"
	"gnd.la/util/generic"
	"gnd.la/util/yaml"
)

type Network struct {
	Name             string
	Description      string
	PublisherExample string `yaml:"publisher_example"`
	URL              string
	Script           string
	RequiresSlot     bool   `yaml:"requires_slot"`
	DefaultSlot      string `yaml:"default_slot"`
	Responsive       bool   `yaml:"responsive"`
	Sizes            []string
}

func main() {
	var networks []Network
	if err := yaml.UnmarshalFile("ads.yaml", &networks); err != nil {
		panic(err)
	}
	generateSizes(networks)
	generateProviders(networks)
}

func sizeName(s string) string {
	return "Size" + s
}

func networkAdSizes(n Network) string {
	adSizes := strings.Join(generic.Map(n.Sizes, sizeName).([]string), ", ")
	if n.Responsive {
		adSizes += ", SizeResponsive"
	}
	return adSizes
}

func generateProviders(networks []Network) {
	var buf bytes.Buffer
	buf.WriteString("package ads\n\n")
	buf.WriteString(genutil.AutogenString())
	buf.WriteString("var (\n")
	for _, n := range networks {
		desc := n.Description
		if desc == "" {
			desc = n.Name
		}
		fmt.Fprintf(&buf, "// %s implements a Provider which displays %s ads.\n", n.Name, desc)
		fmt.Fprintf(&buf, "// For more information, see %s.\n//\n", n.URL)
		fmt.Fprintf(&buf, "// %s supports the following ad sizes:\n", n.Name)
		adSizes := networkAdSizes(n)
		lineSize := 50
		for {
			if lineSize > len(adSizes) || strings.LastIndex(adSizes[:lineSize], " ") < 0 {
				fmt.Fprintf(&buf, "// %s.\n", adSizes)
				break
			}
			pos := strings.LastIndex(adSizes[:lineSize], " ")
			fmt.Fprintf(&buf, "// %s\n", adSizes[:pos])
			adSizes = adSizes[pos+1:]
		}
		fmt.Fprintf(&buf, `%s = &Provider{
	    Name: %q,
	    URL: %q,
	    script: %q,
	    defaultSlot: %q,
	    requiresSlot: %v,
	    responsive: %v,
	    render: render%sAd,
	    className: %q,
	}`, n.Name, n.Name, n.URL, n.Script, n.DefaultSlot, n.RequiresSlot, n.Responsive, n.Name, "ads-"+strings.ToLower(n.Name))
		buf.WriteString("\n")
	}
	buf.WriteString("\nproviders = []*Provider{\n")
	for _, n := range networks {
		fmt.Fprintf(&buf, "%s,\n", n.Name)
	}
	buf.WriteString("\n}")
	buf.WriteString("\n)\n")
	buf.WriteString("func (p *Provider) supportsSize(s Size) bool {\n")
	buf.WriteString("switch p {\n")
	for _, n := range networks {
		fmt.Fprintf(&buf, "case %s:\nreturn ", n.Name)
		for _, s := range n.Sizes {
			fmt.Fprintf(&buf, "s == %s ||", sizeName(s))
		}
		if n.Responsive {
			buf.WriteString("s == SizeResponsive")
		} else {
			buf.Truncate(buf.Len() - 2)
		}
		buf.WriteString("\n")
	}
	buf.WriteString("\n}\nreturn false\n}\n")
	if err := genutil.WriteAutogen("providers_gen.go", buf.Bytes()); err != nil {
		panic(err)
	}
}

func generateSizes(networks []Network) {
	sizeSet := make(map[string]struct{})
	for _, n := range networks {
		for _, s := range n.Sizes {
			sizeSet[s] = struct{}{}
		}
	}
	sizes := generic.Keys(sizeSet).([]string)
	sort.Strings(sizes)
	var buf bytes.Buffer
	buf.WriteString("package ads\n\n")
	buf.WriteString(genutil.AutogenString())
	buf.WriteString("import \"fmt\"\n")
	buf.WriteString("const (\n")
	for _, v := range sizes {
		p := strings.Split(v, "x")
		if len(p) != 2 {
			panic(fmt.Errorf("invalid size %s", v))
		}
		width, err := strconv.Atoi(p[0])
		if err != nil {
			panic(err)
		}
		height, err := strconv.Atoi(p[1])
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(&buf, "%s Size = Size(%d << 16 | %d)\n", sizeName(v), width, height)
	}
	buf.WriteString("\n)\n")
	buf.WriteString("var sizes = map[string]Size{\n")
	for _, v := range sizes {
		fmt.Fprintf(&buf, "%q: %s,\n", "S"+v, sizeName(v))
	}
	fmt.Fprintf(&buf, "%q: %s,\n", "Responsive", sizeName("Responsive"))
	buf.WriteString("\n}\n")
	buf.WriteString("func (s Size) String() string{\n")
	buf.WriteString("switch s {\n")
	for _, v := range sizes {
		fmt.Fprintf(&buf, "case %s:\n return %q\n", sizeName(v), v)
	}
	buf.WriteString("case SizeResponsive:\nreturn \"responsive\"\n")
	buf.WriteString("\n}\nreturn fmt.Sprintf(\"invalid size %d\", int(s))}\n")
	if err := genutil.WriteAutogen("sizes_gen.go", buf.Bytes()); err != nil {
		panic(err)
	}
}
