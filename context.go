package https

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"html"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
)

type Context struct {
	W http.ResponseWriter
	R *http.Request
}

func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{W: w, R: r}
}

func (p *Context) NotFound() {
	http.NotFound(p.W, p.R)
}

func (p *Context) ServeFile(name string) {
	http.ServeFile(p.W, p.R, name)
}

func (p *Context) FormValue(name string) string {
	return p.R.FormValue(name)
}

func (p *Context) Form() url.Values {
	p.R.ParseForm()

	return p.R.Form
}

func (p *Context) PostForm() url.Values {
	p.R.ParseForm()

	return p.R.PostForm
}

//read value from URL or form
func (p *Context) ReadQuery(data interface{}) error {
	if data == nil {
		return errDataType
	}

	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Ptr {
		return errDataType
	}

	v = reflect.Indirect(v)
	t := v.Type()
	r := p.R
	r.ParseForm()
	form := r.Form

	for i := 0; i < v.NumField(); i++ {
		fieldT := t.Field(i)
		fieldTag := fieldT.Tag.Get("url")
		if fieldTag == "" || fieldTag == "-" {
			continue
		}

		if f := form[fieldTag]; f != nil && f[0] != "" {
			vf := v.Field(i)
			s := f[0]

			switch vf.Kind() {

			case reflect.String:
				vf.SetString(s)

			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if n, err := strconv.ParseInt(s, 10, 64); err == nil {
					vf.SetInt(n)
				} else {
					return err
				}

			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				if n, err := strconv.ParseUint(s, 10, 64); err == nil {
					vf.SetUint(n)
				} else {
					return err
				}

			case reflect.Float32, reflect.Float64:
				if n, err := strconv.ParseFloat(s, fieldT.Type.Bits()); err == nil {
					v.SetFloat(n)
				} else {
					return err
				}
			case reflect.Bool:
				if n, err := strconv.ParseBool(s); err == nil {
					v.SetBool(n)
				} else {
					return err
				}

			default:
				return errUnknownDataType
			}

		}
	}

	return nil
}

//add header to the response.
func (p *Context) AddHeader(name, value string) {
	p.W.Header().Add(name, value)
}

//set header
func (p *Context) SetHeader(name, value string) {
	p.W.Header().Set(name, value)
}

//del response header
func (p *Context) DelHeader(name string) {
	p.W.Header().Del(name)
}

//h may be nil
func (p *Context) WriteHeader(statusCode int, h http.Header) {
	if h != nil {
		hto := p.W.Header()
		for k, v := range h {
			hto[k] = v
		}
	}
	p.W.WriteHeader(statusCode)
}

func (p *Context) Write(data []byte) (n int, err error) {
	return p.W.Write(data)
}

func (p *Context) WriteString(str string) error {
	_, err := p.W.Write([]byte(html.EscapeString(str)))

	return err
}

func (p *Context) WriteJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err == nil {
		var buf bytes.Buffer
		json.HTMLEscape(&buf, data)
		p.SetHeader(headerTypeContentType, headerTypeContentJSON)
		_, err = buf.WriteTo(p.W)
	}

	return err
}

func (p *Context) WriteXML(v interface{}) error {
	data, err := xml.Marshal(v)
	if err == nil {
		p.SetHeader(headerTypeContentType, headerTypeContentXML)
		_, err = p.W.Write(data)
	}

	return err
}

var (
	errUnknownDataType = errors.New("Unknown data type")
	errDataType        = errors.New("Error data type")
)

const (
	headerTypeContentType = "Content-Type"
	headerTypeContentJSON = "application/json;charset=utf-8"
	headerTypeContentXML  = "text/xml;charset=utf-8"
)
