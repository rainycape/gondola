package i18n

import (
	"errors"
	"strconv"
)

const (
	// These are the possible values for *strconv.NumError
	_ String = "value out of range"
	_ String = "invalid syntax"
)

// Error represents an error which can be translated to
// another language. Keep in mind that Error also implements
// error, so you can return an Error from any function that
// returns error. You can later use FromError() to get back
// an Error again or TranslatedError, to get an error with
// the translated message.
type Error interface {
	// Error returns the untranslated error message.
	Error() string
	// Err translates the Error and returns it as an error.
	Err(languager Languager) error
	// TranslatedError returns the translated error message.
	TranslatedError(languager Languager) string
}

// translatableError implements the Error interface.
type translatableError struct {
	Format string
	Args   []interface{}
}

func (e *translatableError) Error() string {
	return Sprintf(e.Format, nil, e.Args...)
}

func (e *translatableError) Err(languager Languager) error {
	return errors.New(e.TranslatedError(languager))
}

func (e *translatableError) TranslatedError(languager Languager) string {
	return Sprintf(e.Format, languager, e.Args...)
}

func (e *translatableError) TranslatedString(languager Languager) string {
	return e.TranslatedError(languager)
}

// Errorf returns a error with the given format and arguments. The
// returned Error uses Sprintf internally, which means it will
// translate any argument which supports translation.
func Errorf(format string, args ...interface{}) Error {
	return &translatableError{format, args}
}

// NewError returns an Error with the given message.
func NewError(message string) Error {
	return &translatableError{Format: message}
}

// FromError returns an Error from an error, translating
// it when possible. If e already implements Error, the received
// value is returned.
func FromError(e error) Error {
	if err, ok := e.(Error); ok {
		return err
	}
	if err, ok := e.(*strconv.NumError); ok {
		return Errorf("could not parse %q: %s", err.Num, String(err.Err.Error()))
	}
	return NewError(e.Error())
}

func TranslatedError(err error, languager Languager) error {
	terr := FromError(err)
	if terr != nil {
		return terr.Err(languager)
	}
	return nil
}
