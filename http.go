package khc

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"git.kanosolution.net/kano/kaos"
	"git.kanosolution.net/kano/kaos/client"
	"github.com/sebarcode/codekit"
)

const (
	ReturnResponse string = "RESPONSE"
	ReturnBytes           = "BYTES"
	ReturnObject          = "OBJECT"

	KeyContentType  string = "HttpContentType"
	KeyMethod       string = "HttpMethod"
	KeyReturnType   string = "HttpReturnType"
	KeyReferenceObj string = "HttpReferenceObj"
)

var (
	DefaultReturnType = ReturnObject
)

type HttpClient struct {
	client.ClientBase
	baseUrl string
}

func (h *HttpClient) Close() {
	//panic("not implemented") // TODO: Implement
}

func (h *HttpClient) Call(name string, ref interface{}, data interface{}, configs ...codekit.M) (interface{}, error) {
	var e error
	if ref == nil {
		return nil, errors.New("reference object is missing")
	}

	var refPtr interface{}
	isPtr := false
	v := reflect.ValueOf(ref)
	if v.Kind() == reflect.Ptr {
		isPtr = true
		refPtr = ref
	} else {
		refPtr = reflect.New(v.Type()).Interface()
	}
	e = h.CallTo(name, refPtr, data, configs...)
	if e != nil {
		return ref, e
	}
	if isPtr {
		return refPtr, nil
	} else {
		return reflect.ValueOf(refPtr).Elem().Interface(), nil
	}
}

func (h *HttpClient) CallTo(name string, target interface{}, parm interface{}, configs ...codekit.M) error {
	var e error
	config := kaos.MergeMs(configs...)
	callType := config.GetString(KeyMethod)
	if callType == "" {
		callType = http.MethodPost
	}
	contentType := config.GetString(KeyContentType)
	if contentType == "" {
		contentType = "application/json"
	}

	uri := h.baseUrl + name
	bs, e := h.Byter().Encode(parm)
	if e != nil {
		return fmt.Errorf("fail to encode parm. %s", e.Error())
	}

	var (
		resp *http.Response
		req  *http.Request
	)
	req, e = http.NewRequest(callType, uri, bytes.NewBuffer(bs))
	if e != nil {
		return fmt.Errorf("fail to create http request. %s", e.Error())
	}
	req.Header.Set("Content-Type", contentType)
	hc := http.Client{}
	resp, e = hc.Do(req)
	if e != nil {
		return fmt.Errorf("fail to create call %s. %s", uri, e.Error())
	}

	bs, e = ioutil.ReadAll(resp.Body)
	if e != nil {
		return fmt.Errorf("fail to read result %s. %s", uri, e.Error())
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("fail to create call %s. %s - %s", uri, resp.Status, string(bs))
	}

	e = h.This().Byter().DecodeTo(bs, target, nil)
	if e != nil {
		return fmt.Errorf("fail to decode result %s. %s", uri, e.Error())
	}
	return nil
}

func init() {
	client.RegisterClient("http", NewHttpClient)
}

// NewHttpClient call new client
func NewHttpClient(host string, config codekit.M) (client.Client, error) {
	c := new(HttpClient)
	protocols := strings.Split(host, "://")
	hasProtocols := len(protocols) > 1
	c.baseUrl = host
	if strings.HasSuffix(c.baseUrl, "/") {
		c.baseUrl = c.baseUrl[0 : len(c.baseUrl)-1]
	}
	if !hasProtocols {
		// test with http
		if _, e := http.Get("http://" + c.baseUrl); e == nil {
			c.baseUrl = "http://" + c.baseUrl
			return c, nil
		}

		if _, e := http.Get("https://" + c.baseUrl); e == nil {
			c.baseUrl = "https://" + c.baseUrl
			return c, nil
		}
		return nil, fmt.Errorf("unreached server http(s)://%s", c.baseUrl)
	}

	c.SetThis(c)
	return c, nil
}
