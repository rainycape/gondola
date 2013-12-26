package formutil

import (
	"fmt"
	"gnd.la/app"
	"gnd.la/form"
	"gnd.la/html"
	"gnd.la/i18n"
	"math/rand"
	"strconv"
)

// MathCaptcha provides a simple captcha
// which presents a simple arithmetic operation
// to be solved by the user. This won't protect
// you from targetted attacks, but it will stop
// most spam bots.
func MathCaptcha() interface{} {
	return &mathCaptcha{
		MathCaptchaA: rand.Int() % 10,
		MathCaptchaB: rand.Int() % 10,
	}
}

type mathCaptcha struct {
	MathCaptchaA  int    `form:",hidden"`
	MathCaptchaB  int    `form:",hidden"`
	CaptchaResult string `form:",optional,max_length=2,label=formutil|Are you human?,help=formutil|This is used to prevent spam,placeholder=formutil|Result"`
}

func (s *mathCaptcha) ValidateCaptchaResult() error {
	if s.CaptchaResult == "" {
		return i18n.Errorfc("formutil", "please, enter the result of the operation")
	}
	r := -1
	p, err := strconv.Atoi(s.CaptchaResult)
	if err == nil {
		r = p
	}
	if r != s.MathCaptchaA+s.MathCaptchaB {
		return i18n.Errorfc("formutil", "incorrect result")
	}
	return nil
}

func (s *mathCaptcha) FieldAddOns(ctx *app.Context, field *form.Field) []*form.AddOn {
	if field.GoName == "CaptchaResult" {
		return []*form.AddOn{
			&form.AddOn{
				Node:     html.Text(fmt.Sprintf("%d+%d =", s.MathCaptchaA, s.MathCaptchaB)),
				Position: form.AddOnPositionBefore,
			},
		}
	}
	return nil
}
