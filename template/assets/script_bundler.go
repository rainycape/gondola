package assets

import (
	"encoding/json"
	"fmt"
	"gnd.la/log"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

// ScriptBundler uses Google's closure compiler to optimize JS code.
// Accepted options are:
//  optimize: (simple|advanced) - defaults to simple
//  compiler_warnings: boolean - defaults to false
type scriptBundler struct {
}

func (c *scriptBundler) Bundle(w io.Writer, r io.Reader, opts Options) error {
	code, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	level := "SIMPLE_OPTIMIZATIONS"
	if opts.StringOpt("optimize") == "advanced" {
		level = "ADVANCED_OPTIMIZATIONS"
	}
	outputInfo := []string{"compiled_code", "errors", "statistics"}
	if opts.BoolOpt("compiler_warnings") {
		outputInfo = append(outputInfo, "warnings")
	}
	form := url.Values{
		"js_code":           []string{string(code)},
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
		return fmt.Errorf("server errors: %v", se)
	}
	if e := out["errors"]; e != nil {
		return fmt.Errorf("errors: %v", e)
	}
	if w := out["warning"]; w != nil {
		log.Warningf("warnings when compiling code: %v", w)
	}
	if stats := out["statistics"]; stats != nil {
		if statsm, _ := stats.(map[string]interface{}); statsm != nil {
			log.Debugf("Compressed JS from %v to %v (GZIP'ed %v to %v)",
				statsm["originalSize"], statsm["compressedSize"],
				statsm["originalGzipSize"], statsm["compressedGzipSize"])
		}
	}
	compiled := out["compiledCode"].(string)
	_, err = io.WriteString(w, compiled)
	return err
}

func (c *scriptBundler) Type() Type {
	return TypeJavascript
}

func init() {
	RegisterBundler(&scriptBundler{})
}
