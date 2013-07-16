package assets

import (
	"fmt"
	"gondola/hashutil"
	"gondola/log"
	"path"
	"path/filepath"
	"strings"
)

const (
	// If you define your own code types, use numbers > 1000
	CodeTypeCss        = 1
	CodeTypeJavascript = 2
)

type CodeAsset interface {
	Asset
	CodeType() int
	Code() (string, error)
}

type CodeAssetList []CodeAsset

func (c CodeAssetList) Names() []string {
	var names []string
	for _, v := range c {
		names = append(names, v.Name())
	}
	return names
}

func (c CodeAssetList) CompiledName(ext string, o Options) string {
	if len(c) == 0 {
		return ""
	}
	code, _ := Code(c)
	h := hashutil.Fnv32a(code + o.String())
	name := c[0].Name()
	if ext == "" {
		ext = filepath.Ext(name)
	} else {
		ext = "." + ext
	}
	return path.Join(filepath.Dir(name), "t-"+h+ext)
}

type Compiler func(Manager, []CodeAsset, Options) ([]Asset, error)

var (
	compilers = map[int]Compiler{}
)

func RegisterCompiler(c Compiler, codeType int) {
	registerCompiler(c, codeType, false)
}

func registerCompiler(c Compiler, codeType int, internal bool) {
	// Let users override the default compilers
	if !internal && codeType <= 1000 && compilers[codeType] == nil {
		panic(fmt.Errorf("Invalid custom code type %d (must be > 1000)", codeType))
	}
	compilers[codeType] = c
}

func CodeAssets(assets []Asset) []CodeAsset {
	var codeAssets []CodeAsset
	for _, v := range assets {
		if c, ok := v.(CodeAsset); ok {
			codeAssets = append(codeAssets, c)
		}
	}
	return codeAssets
}

func Code(assets []CodeAsset) (string, error) {
	var codes []string
	for _, v := range assets {
		c, err := v.Code()
		if err != nil {
			return "", err
		}
		codes = append(codes, c)
	}
	return strings.Join(codes, "\n"), nil
}

func Compile(m Manager, assets []Asset, o Options) ([]Asset, error) {
	if len(assets) == 0 {
		return nil, nil
	}
	dirs := make(map[string]struct{})
	exts := make(map[string]struct{})
	for _, v := range assets {
		dirs[filepath.Dir(v.Name())] = struct{}{}
		exts[filepath.Ext(v.Name())] = struct{}{}
	}
	if len(dirs) > 1 {
		var d []string
		for k, _ := range dirs {
			d = append(d, k)
		}
		log.Warningf("Compiling assets from different directories (%s), relative links might not work.", strings.Join(d, ", "))
	}
	if len(exts) > 1 {
		var e []string
		for k, _ := range exts {
			e = append(e, k)
		}
		log.Warningf("Compiling assets with different extensions (%s).", strings.Join(e, ", "))
	}
	codeAssets := CodeAssets(assets)
	if len(codeAssets) != len(assets) {
		return nil, fmt.Errorf("Some assets don't implement CodeAsset")
	}
	ctype := codeAssets[0].CodeType()
	compiler := compilers[ctype]
	if compiler == nil {
		return nil, fmt.Errorf("No compiler for code type %d", ctype)
	}
	log.Debugf("Compiling %v", CodeAssetList(codeAssets).Names())
	return compiler(m, codeAssets, o)
}
