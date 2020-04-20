package plist

import "errors"

type optionReceiver interface {
	unmarshalerSetLax(bool) (bool, error)
	generatorSetGNUStepBase64(bool) (bool, error)
	generatorSetIndent(string) (bool, error)
	encoderSetFormat(int) (bool, error)
}

type Option func(optionReceiver) (bool, error)

var optionInvalidError = errors.New("this option is unsupported for this format")

func GNUStepUseBase64Data() Option {
	return Option(func(o optionReceiver) (bool, error) {
		return o.generatorSetGNUStepBase64(true)
	})
}

func Indent(i string) Option {
	return Option(func(o optionReceiver) (bool, error) {
		return o.generatorSetIndent(i)
	})
}

func Format(f int) Option {
	return Option(func(o optionReceiver) (bool, error) {
		return o.encoderSetFormat(f)
	})
}
