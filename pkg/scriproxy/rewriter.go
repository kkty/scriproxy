package scriproxy

import (
	"net/http"
	"net/url"

	"github.com/d5/tengo/compiler/token"
	"github.com/d5/tengo/objects"
	"github.com/d5/tengo/script"
	"github.com/d5/tengo/stdlib"
)

type funcBase struct {
	name string
}

func (o *funcBase) TypeName() string {
	return o.name
}

func (o *funcBase) String() string {
	return o.name
}

func (o *funcBase) BinaryOp(op token.Token, rhs objects.Object) (objects.Object, error) {
	return nil, nil
}

func (o *funcBase) IsFalsy() bool {
	return false
}

func (o *funcBase) Equals(another objects.Object) bool {
	return false
}

func (o *funcBase) Copy() objects.Object {
	return nil
}

type headerGetFunc struct {
	*funcBase
	header http.Header
}

func (o *headerGetFunc) Call(args ...objects.Object) (ret objects.Object, err error) {
	if len(args) != 1 {
		return nil, objects.ErrWrongNumArguments
	}

	s, ok := objects.ToString(args[0])

	if !ok {
		return nil, objects.ErrInvalidArgumentType{}
	}

	return objects.FromInterface(o.header.Get(s))
}

func newHeaderGetFunc(header http.Header) *headerGetFunc {
	return &headerGetFunc{
		&funcBase{"headerGetFunc"},
		header,
	}
}

type headerSetFunc struct {
	*funcBase
	header http.Header
}

func (o *headerSetFunc) Call(args ...objects.Object) (ret objects.Object, err error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	k, ok := objects.ToString(args[0])

	if !ok {
		return nil, objects.ErrInvalidArgumentType{}
	}

	v, ok := objects.ToString(args[1])

	if !ok {
		return nil, objects.ErrInvalidArgumentType{}
	}

	o.header.Set(k, v)

	return objects.UndefinedValue, nil
}

func newHeaderSetFunc(header http.Header) *headerSetFunc {
	return &headerSetFunc{
		&funcBase{"headerSetFunc"},
		header,
	}
}

type queryGetFunc struct {
	*funcBase
	values url.Values
}

func (o queryGetFunc) Call(args ...objects.Object) (ret objects.Object, err error) {
	if len(args) != 1 {
		return nil, objects.ErrWrongNumArguments
	}

	k, ok := objects.ToString(args[0])

	if !ok {
		return nil, objects.ErrInvalidArgumentType{}
	}

	return objects.FromInterface(o.values.Get(k))
}

func newQueryGetFunc(values url.Values) *queryGetFunc {
	return &queryGetFunc{
		&funcBase{"queryGetFunc"},
		values,
	}
}

type querySetFunc struct {
	*funcBase
	values url.Values
}

func (o querySetFunc) Call(args ...objects.Object) (ret objects.Object, err error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	k, ok := objects.ToString(args[0])

	if !ok {
		return nil, objects.ErrInvalidArgumentType{}
	}

	v, ok := objects.ToString(args[1])

	if !ok {
		return nil, objects.ErrInvalidArgumentType{}
	}

	o.values.Set(k, v)

	return objects.UndefinedValue, nil
}

func newQuerySetFunc(values url.Values) *querySetFunc {
	return &querySetFunc{
		&funcBase{"querySetFunc"},
		values,
	}
}

// RequestRewriter can be used to rewrite http requests.
type RequestRewriter struct {
	compiledScript *script.Compiled
}

// NewRequestRewriter creates a RequestRewriter out of a tengo script.
// The libraries used in the script should also be specified.
func NewRequestRewriter(
	userScript []byte,
	libraries []string,
) (RequestRewriter, error) {
	s := script.New(userScript)

	for _, library := range libraries {
		s.SetImports(stdlib.GetModuleMap(library))
	}

	if err := s.Add("req", map[string]interface{}{
		"host": "",
		"url": map[string]interface{}{
			"scheme": "",
			"host":   "",
			"path":   "",
			"query": map[string]interface{}{
				"get": newQueryGetFunc(nil),
				"set": newQuerySetFunc(nil),
			},
		},
		"header": map[string]interface{}{
			"get": newHeaderGetFunc(nil),
			"set": newHeaderSetFunc(nil),
		},
	}); err != nil {
		return RequestRewriter{}, err
	}

	compiled, err := s.Compile()

	if err != nil {
		return RequestRewriter{}, err
	}

	return RequestRewriter{compiled}, nil
}

// Rewrite accepts a pointer to http.Request and rewrites it.
func (w RequestRewriter) Rewrite(request *http.Request) error {
	compiledScript := w.compiledScript.Clone()

	query := request.URL.Query()

	requestObject := map[string]interface{}{
		"host": request.Host,
		"url": map[string]interface{}{
			"scheme": request.URL.Scheme,
			"host":   request.URL.Host,
			"path":   request.URL.Path,
			"query": map[string]interface{}{
				"get": newQueryGetFunc(query),
				"set": newQuerySetFunc(query),
			},
		},
		"header": map[string]interface{}{
			"get": newHeaderGetFunc(request.Header),
			"set": newHeaderSetFunc(request.Header),
		},
	}

	if err := compiledScript.Set("req", requestObject); err != nil {
		return err
	}

	if err := compiledScript.Run(); err != nil {
		return err
	}

	requestObject = compiledScript.Get("req").Map()

	url := requestObject["url"].(map[string]interface{})
	request.URL.Scheme = url["scheme"].(string)
	request.URL.Path = url["path"].(string)
	request.URL.Host = url["host"].(string)

	request.URL.RawQuery = query.Encode()

	request.Host = requestObject["host"].(string)

	return nil
}
