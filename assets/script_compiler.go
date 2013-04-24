package assets

import (
	"encoding/json"
	"fmt"
	"gondola/log"
	"net/http"
	"net/url"
)

// ScriptCompiler uses Google's closure compiler to optimize JS code.
// Accepted options are:
//  optimize: (simple|advanced) - defaults to simple
//  compiler_warnings: boolean - defaults to false
func scriptCompiler(m Manager, name string, assets []CodeAsset, o Options) error {
	code, err := Code(assets)
	if err != nil {
		return err
	}
	level := "SIMPLE_OPTIMIZATIONS"
	if o.StringOpt("optimize", m) == "advanced" {
		level = "ADVANCED_OPTIMIZATIONS"
	}
	outputInfo := []string{"compiled_code", "errors", "statistics"}
	if o.BoolOpt("compiler_warnings", m) {
		outputInfo = append(outputInfo, "warnings")
	}
	form := url.Values{
		"js_code":           []string{code},
		"compilation_level": []string{level},
		"output_format":     []string{"json"},
		"output_info":       outputInfo,
	}
	resp, err := http.PostForm("http://closure-compiler.appspot.com/compile", form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var out map[string]interface{}
	err = decoder.Decode(&out)
	if err != nil {
		return err
	}
	if se := out["serverErrors"]; se != nil {
		return fmt.Errorf("Server errors: %v", se)
	}
	if e := out["errors"]; e != nil {
		return fmt.Errorf("Errors: %v", e)
	}
	if w := out["warning"]; w != nil {
		log.Warningf("Warnings when compiling code: %v", w)
	}
	compiled := out["compiledCode"].(string)
	f, err := m.Create(name)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(compiled))
	if err != nil {
		return err
	}
	f.Close()
	if stats := out["statistics"]; stats != nil {
		if statsm, _ := stats.(map[string]interface{}); statsm != nil {
			log.Debugf("Compressed code from %v to %v (GZIP'ed %v to %v)",
				statsm["originalSize"], statsm["compressedSize"],
				statsm["originalGzipSize"], statsm["compressedGzipSize"])
		}
	}
	return nil
}

func ScriptCompiler(m Manager, assets []CodeAsset, o Options) ([]Asset, error) {
	name := CodeAssetList(assets).CompiledName("", o)
	_, _, err := m.Load(name)
	if err != nil {
		err = scriptCompiler(m, name, assets, o)
		if err != nil {
			return nil, err
		}
	}
	return ScriptParser(m, []string{name}, o)
}

func init() {
	registerCompiler(ScriptCompiler, CodeTypeJavascript, true)
}
