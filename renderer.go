package smutje

import (
	"bytes"
	"io"
	"io/ioutil"
	"text/template"

	"github.com/pkg/errors"
)

func renderString(context, input string, attrs Attributes) (string, error) {
	tmpl, err := template.New(context).Parse(input)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse template")
	}
	tmpl.Option("missingkey=error")
	buf := bytes.NewBuffer(nil)
	err = tmpl.Execute(buf, attrs)
	return buf.String(), errors.Wrap(err, "failed to render template")
}

func renderFile(filename string, attrs Attributes) (io.ReadCloser, error) {
	tpl, err := template.ParseFiles(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse template")
	}
	tpl.Option("missingkey=error")
	buf := bytes.NewBuffer(nil)
	err = tpl.Execute(buf, attrs)
	return ioutil.NopCloser(buf), errors.Wrap(err, "failed to render template")
}
