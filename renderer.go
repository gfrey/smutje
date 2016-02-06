package smutje

import (
	"bytes"
	"io"
	"io/ioutil"
	"text/template"
)

func renderString(context, input string, attrs smAttributes) (string, error) {
	tmpl, err := template.New(context).Parse(input)
	if err != nil {
		return "", err
	}
	tmpl.Option("missingkey=error")
	buf := bytes.NewBuffer(nil)
	err = tmpl.Execute(buf, attrs)
	return buf.String(), err
}

func renderFile(filename string, attrs smAttributes) (io.ReadCloser, error) {
	tpl, err := template.ParseFiles(filename)
	if err != nil {
		return nil, err
	}
	tpl.Option("missingkey=error")
	buf := bytes.NewBuffer(nil)
	if err := tpl.Execute(buf, attrs); err != nil {
		return nil, err
	}
	return ioutil.NopCloser(buf), nil
}
