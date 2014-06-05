package form

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"strings"

	"gnd.la/app"
	"gnd.la/crypto/cryptoutil"
	"gnd.la/i18n"
	"gnd.la/util/stringutil"
)

const (
	csrfSalt          = "gnd.la/form.csrf-salt"
	randomSaltLength  = 32
	randomValueLength = 64
)

// csrf implements CSRF protection using signed and encrypted values,
// following the given algorithm.
//
// 1 - Generate two random strings, join them by ':' and encrypt-sign them
// with the App EncryptSigner using csrfSalt to obtain CSRFA.
// 2 - Generate another random string. EncryptSign using the first random string
// in step 1 as salt to obtain CSRFB
// 3 - Reverse random string in step 2. EncryptSign using the second random string
// in step 2 as salt to obtain CSRFC.
//
// Note: In order to avoid requiring an app encryption key, if the app lacks it,
// we create a secret by hashing the app secret.
type csrf struct {
	GondolaCSRFA string `form:",hidden"`
	GondolaCSRFB string `form:",hidden"`
	GondolaCSRFC string `form:",hidden"`
	failed       bool
	salt1        string
	salt2        string
	value        string
}

func (c *csrf) ValidateGondolaCSRFA(ctx *app.Context) error {
	c.failed = false
	es, err := csrfEncryptSigner(ctx, csrfSalt)
	if err != nil {
		return c.error(ctx, err)
	}
	val, err := es.UnsignDecrypt(c.GondolaCSRFA)
	if err != nil {
		return c.error(ctx, err)
	}
	parts := strings.Split(string(val), ":")
	if len(parts) != 2 {
		return c.error(ctx, nil)
	}
	if len(parts[0]) < randomSaltLength || len(parts[1]) < randomSaltLength {
		return c.error(ctx, nil)
	}
	c.salt1 = parts[0]
	c.salt2 = parts[1]
	return nil
}

func (c *csrf) ValidateGondolaCSRFB(ctx *app.Context) error {
	if c.failed {
		return nil
	}
	es, err := csrfEncryptSigner(ctx, c.salt1)
	if err != nil {
		return c.error(ctx, err)
	}
	val, err := es.UnsignDecrypt(c.GondolaCSRFB)
	if err != nil {
		return c.error(ctx, err)
	}
	c.value = string(val)
	if len(c.value) < randomValueLength {
		return c.error(ctx, nil)
	}
	return nil
}

func (c *csrf) ValidateGondolaCSRFC(ctx *app.Context) error {
	if c.failed {
		return nil
	}
	es, err := csrfEncryptSigner(ctx, c.salt2)
	if err != nil {
		return c.error(ctx, err)
	}
	val, err := es.UnsignDecrypt(c.GondolaCSRFC)
	if err != nil {
		return c.error(ctx, err)
	}
	rev := stringutil.Reverse(string(val))
	if len(rev) != len(c.value) || subtle.ConstantTimeCompare([]byte(c.value), []byte(rev)) != 1 {
		return c.error(ctx, nil)
	}
	return nil
}

func (c *csrf) generate(ctx *app.Context) (*csrf, error) {
	salt1 := stringutil.Random(randomSaltLength)
	salt2 := stringutil.Random(randomSaltLength)
	esA, err := csrfEncryptSigner(ctx, csrfSalt)
	if err != nil {
		return nil, err
	}
	c.GondolaCSRFA, err = esA.EncryptSign([]byte(salt1 + ":" + salt2))
	if err != nil {
		return nil, err
	}
	value := stringutil.Random(randomValueLength)
	esB, err := csrfEncryptSigner(ctx, salt1)
	if err != nil {
		return nil, err
	}
	c.GondolaCSRFB, err = esB.EncryptSign([]byte(value))
	if err != nil {
		return nil, err
	}
	esC, err := csrfEncryptSigner(ctx, salt2)
	if err != nil {
		return nil, err
	}
	c.GondolaCSRFC, err = esC.EncryptSign([]byte(stringutil.Reverse(value)))
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *csrf) error(ctx *app.Context, err error) error {
	c.failed = true
	if _, err := c.generate(ctx); err != nil {
		panic(err)
	}
	return i18n.NewError("invalid CSRF token - please, submit the form again").Err(ctx)
}

func csrfEncryptSigner(ctx *app.Context, salt string) (*cryptoutil.EncryptSigner, error) {
	a := ctx.App()
	signer, err := a.Signer([]byte(salt))
	if err != nil {
		return nil, err
	}
	encrypter, _ := a.Encrypter()
	if encrypter == nil {
		var key []byte
		cfg := a.Config()
		if cfg == nil || cfg.Secret == "" {
			return nil, errors.New("can't generate CSRF tokens, App configuration has no Secret")
		}
		if len(cfg.EncryptionKey) > 0 {
			key = []byte(cfg.EncryptionKey)
		} else {
			s := sha256.New()
			s.Write([]byte(cfg.Secret))
			key = s.Sum(nil)
		}
		encrypter = &cryptoutil.Encrypter{Cipherer: a.Cipherer, Key: key}
	}
	return &cryptoutil.EncryptSigner{Encrypter: encrypter, Signer: signer}, nil
}

func newCSRF(f *Form) (*csrf, error) {
	c := &csrf{}
	if !f.Submitted() {
		return c.generate(f.ctx)
	}
	c.GondolaCSRFA = f.ctx.FormValue(f.toHTMLName("GondolaCSRFA"))
	c.GondolaCSRFB = f.ctx.FormValue(f.toHTMLName("GondolaCSRFB"))
	c.GondolaCSRFC = f.ctx.FormValue(f.toHTMLName("GondolaCSRFC"))
	return c, nil
}
