package https

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/gorilla/schema"
	"html"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
)

type Context struct {
	W      http.ResponseWriter
	R      *http.Request
	MaxMem int64 //upload file memory size
}

func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{W: w, R: r, MaxMem: 0x4000000}
}

func (p *Context) NotFound() {
	http.NotFound(p.W, p.R)
}

//code should be status code in http package, example: http.StatusContinue
func (p *Context) Error(code int) {
	msg := fmt.Sprintf("%d %s", code, http.StatusText(code))
	http.Error(p.W, msg, code)
}

func (p *Context) ServeFile(name string) {
	http.ServeFile(p.W, p.R, name)
}

func (p *Context) FormValue(name string) string {
	return p.R.FormValue(name)
}

func (p *Context) Query() url.Values {
	return p.R.URL.Query()
}

func (p *Context) Form() url.Values {
	p.R.ParseForm()

	return p.R.Form
}

func (p *Context) PostForm() url.Values {
	p.R.ParseForm()

	return p.R.PostForm
}

// ReadForm binds the formObject  with the form data
// it supports any kind of type, including custom structs.
// It will return nothing if request data are empty.
// The struct field tag is "form".
//
func (p *Context) ReadForm(data interface{}) error {
	values := p.Form()
	if len(values) == 0 {
		return nil
	}

	return decoderForm.Decode(data, values)
}

// ReadQuery binds the "ptr" with the url query string. The struct field tag is "url".
func (p *Context) ReadQuery(data interface{}) error {
	values := p.Query()
	if len(values) == 0 {
		return nil
	}

	return decoderQuery.Decode(data, values)
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

func (p *Context) WriteDataJSON(data []byte) error {
	var buf bytes.Buffer
	json.HTMLEscape(&buf, data)
	p.SetHeader(headerTypeContentType, headerTypeContentJSON)
	_, err := buf.WriteTo(p.W)

	return err
}

func (p *Context) WriteJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err == nil {
		err = p.WriteDataJSON(data)
	}

	return err
}

func (p *Context) WriteDataXML(data []byte) error {
	p.SetHeader(headerTypeContentType, headerTypeContentXML)
	_, err := p.W.Write(data)

	return err
}

func (p *Context) WriteXML(v interface{}) error {
	data, err := xml.Marshal(v)
	if err == nil {
		err = p.WriteDataXML(data)
	}

	return err
}

func (p *Context) WriteHTML(data string) error {
	p.SetHeader(headerTypeContentType, headerTypeContentHTML)

	return p.WriteString(data)
}

func (p *Context) WriteText(data string) error {
	p.SetHeader(headerTypeContentType, headerTypeContentText)
	_, err := p.Write([]byte(data))

	return err
}

//read header from request
func (p *Context) GetHeader(name string) string {
	return p.R.Header.Get(name)
}

func (p *Context) GetBody() ([]byte, error) {
	if p.R.Body == nil {
		return nil, errEmptyBody
	} else {
		return ioutil.ReadAll(p.R.Body)
	}
}

func (p *Context) UnmarshalBody(v interface{}, unmarshaler UnmarshalerFunc) error {
	data, err := p.GetBody()
	if err == nil {
		err = unmarshaler(data, v)
	}

	return err
}

func (p *Context) ReadJSON(v interface{}) error {
	return p.UnmarshalBody(v, UnmarshalerFunc(json.Unmarshal))
}

func (p *Context) ReadXML(v interface{}) error {
	return p.UnmarshalBody(v, UnmarshalerFunc(xml.Unmarshal))
}

func (p *Context) ReadHTML() (string, error) {
	var str string
	data, err := p.GetBody()
	if err == nil {
		str = html.UnescapeString(string(data))
	}

	return str, err
}

func (p *Context) ReadText() (string, error) {
	var str string
	data, err := p.GetBody()
	if err == nil {
		str = string(data)
	}

	return str, err
}

func (p *Context) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	if err := p.R.ParseMultipartForm(p.MaxMem); err != nil {
		return nil, nil, err
	}

	return p.R.FormFile(key)
}

func (p *Context) UploadFile(key string, createFile func(string) io.WriteCloser) error {
	file, info, err := p.FormFile(key)
	if err == nil {
		defer file.Close()

		if out := createFile(info.Filename); out != nil {
			defer out.Close()
			_, err = io.Copy(out, file)
		} else {
			err = errCreateFile
		}
	}

	return err
}

func init() {
	decoderForm = schema.NewDecoder()
	decoderQuery = schema.NewDecoder()
	decoderForm.SetAliasTag("form")
	decoderQuery.SetAliasTag("url")
}

type (
	UnmarshalerFunc func(data []byte, v interface{}) error
)

var (
	errUnknownDataType = errors.New("Unknown data type")
	errDataType        = errors.New("Error data type")
	errEmptyBody       = errors.New("Empty body")
	errCreateFile      = errors.New("Create file failed")
	decoderForm        *schema.Decoder
	decoderQuery       *schema.Decoder
)

const (
	headerTypeContentType = "Content-Type"
	headerTypeContentJSON = "application/json;charset=utf-8"
	headerTypeContentXML  = "text/xml;charset=utf-8"
	headerTypeContentHTML = "text/html;charset=utf-8"
	headerTypeContentText = "text/plain;charset=utf-8"
)
