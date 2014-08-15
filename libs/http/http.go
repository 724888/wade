package http

import (
	"encoding/json"
	"fmt"
)

const (
	ArrayBuffer = "arraybuffer"
	Blob        = "blob"
	Document    = "document"
	JSON        = "json"
	Text        = "text"
)

var (
	defaultClient *Client
)

func SetDefaultClient(c *Client) {
	defaultClient = c
}

func DefaultClient() *Client {
	if defaultClient == nil {
		panic("No default client has been set.")
	}
	return defaultClient
}

func Do(request *Request) (resp *Response, err error) {
	resp, err = defaultClient.DoPure(request)
	return
}

type Backend interface {
	Do(*Request) error
}

type ResponseHeaders interface {
	String() string
	Get(string) string
}

type Response struct {
	RawData    interface{}
	Data       string
	Status     string
	StatusCode int
	Type       string
	Headers    ResponseHeaders
}

func (r *Response) Failed() bool {
	return r.StatusCode >= 400
}

func (r *Response) DecodeTo(dest interface{}) error {
	if r.Type != "" && r.Type != Text {
		panic("This response's type must be text to be decoded.")
		return nil
	}

	if r.Failed() {
		panic(fmt.Sprintf("Response failed with status %v, cannot decode.", r.Status))
	}

	err := json.Unmarshal([]byte(r.Data), dest)
	if err != nil {
		panic(err.Error())
	}
	return err
}

func (r *Response) Bool() (b bool) {
	r.DecodeTo(&b)
	return
}

type HttpHeader map[string][]string

// Add adds the key, value pair to the header.
// It appends to any existing values associated with key.
func (h HttpHeader) Add(key, value string) {
	if _, ok := h[key]; !ok {
		h[key] = make([]string, 0)
	}
	h[key] = append(h[key], value)
}

// Set sets the header entries associated with key to
// the single element value.  It replaces any existing
// values associated with key.
func (h HttpHeader) Set(key, value string) {
	if _, ok := h[key]; ok {
		h[key] = nil
	}
	h[key] = make([]string, 0)
	h[key] = append(h[key], value)
}

// Get gets the first value associated with the given key.
// If there are no values associated with the key, Get returns "".
func (h HttpHeader) Get(key string) string {
	if v, ok := h[key]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

// Del deletes the values associated with key.
func (h HttpHeader) Del(key string) {
	if _, ok := h[key]; ok {
		delete(h, key)
	}
}

type Request struct {
	data            interface{}
	Headers         HttpHeader
	Method          string
	Url             string
	Response        *Response
	ResponseType    string
	Timeout         int
	WithCredentials bool
}

func NewRequest(method string, rurl string) *Request {
	return &Request{
		Method:  method,
		Headers: make(map[string][]string),
		Url:     rurl,
	}
}

func (r *Request) SetRawData(data interface{}) {
	r.data = data
}

func (r *Request) SetData(d interface{}) (err error) {
	var data []byte
	data, err = json.Marshal(d)
	r.data = string(data)
	return err
}

func (r *Request) Data() string {
	if data, ok := r.data.(string); ok {
		return data
	}

	return ""
}

func (r *Request) RawData() interface{} {
	return r.data
}

type Client struct {
	backend   Backend
	reqInts   []RequestInterceptor
	respInts  []ResponseInterceptor
	blockChan chan bool
}

// Do does an asynchronous http request, yet the API is blocking, just like Go's http
func (c *Client) Do(r *Request) (resp *Response, err error) {
	resp, err = c.DoPure(r)
	c.triggerResponseInterceptors(r)

	return
}

// DoPure performs a request without applying interceptors
func (c *Client) DoPure(r *Request) (resp *Response, err error) {
	//gopherjs:blocking
	err = c.backend.Do(r)
	resp = r.Response
	return
}

type RequestInterceptor func(*Request)
type ResponseInterceptor func(chan bool, *Request)

func NewClient(backend Backend) *Client {
	return &Client{
		backend:   backend,
		reqInts:   make([]RequestInterceptor, 0),
		respInts:  make([]ResponseInterceptor, 0),
		blockChan: make(chan bool, 1),
	}
}

func (c *Client) AddRequestInterceptor(hi RequestInterceptor) {
	c.reqInts = append(c.reqInts, hi)
}

func (c *Client) AddResponseInterceptor(hi ResponseInterceptor) {
	c.respInts = append(c.respInts, hi)
}

func (c *Client) NewRequest(method string, url string) *Request {
	r := NewRequest(method, url)
	for _, intrFn := range c.reqInts {
		intrFn(r)
	}
	return r
}

func (c *Client) GET(url string) (resp *Response, err error) {
	resp, err = c.Do(c.NewRequest("GET", url))
	return
}

func (c *Client) POST(url string, data interface{}) (resp *Response, err error) {
	r := c.NewRequest("POST", url)
	r.SetData(data)
	resp, err = c.Do(r)
	return
}

func (c *Client) triggerResponseInterceptors(r *Request) {
	c.blockChan <- true //Prevent other interceptor handling running concurrently
	finishChannel := make(chan bool, 1)
	for _, intrFn := range c.respInts {
		go func() {
			//gopherjs:blocking
			intrFn(finishChannel, r)
		}()
		<-finishChannel
	}
	<-c.blockChan
}
