package postmangen

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/rbretecher/go-postman-collection"
)

type PostmanGen struct {
	collection          *postman.Collection
	placeholderDefaults map[string]string
}

func NewPostmanGen(name string, description string) *PostmanGen {
	p := &PostmanGen{
		collection:          postman.CreateCollection(name, description),
		placeholderDefaults: map[string]string{},
	}
	p.collection.Auth = postman.CreateAuth(postman.Bearer, &postman.AuthParam{
		Key:   "token",
		Value: "{{token}}",
		Type:  "string",
	})

	return p
}

func walkStructFields(t reflect.Type, fn func(field reflect.StructField)) {
	// ptr -> elem
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		if f.PkgPath != "" {
			continue
		}

		if f.Anonymous {
			ft := f.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				walkStructFields(ft, fn)
				continue
			}
		}

		fn(f)
	}
}

func (p *PostmanGen) AddVariable(key string, value string) *PostmanGen {
	p.collection.Variables = append(p.collection.Variables, &postman.Variable{
		Key:   key,
		Type:  "string",
		Value: value,
	})
	return p
}

func (p *PostmanGen) AddPlaceholder(key string, value string) *PostmanGen {
	p.placeholderDefaults[key] = value
	return p
}

func (p *PostmanGen) Register(spec map[string]any) error {
	defer func() {
		recover()
	}()

	method, ok1 := spec["method"].(string)
	path, ok2 := spec["path"].(string)
	inputType, ok3 := spec["inputType"].(reflect.Type)

	if !ok1 || !ok2 || !ok3 {
		return errors.New("invalid spec: must contain method, path, and inputType")
	}

	typ := inputType
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return errors.New("invalid object type: must be a struct or pointer to struct")
	}

	type FormParam struct {
		Key         string `json:"key"`
		Value       string `json:"value"`
		Type        string `json:"type"` // text | file
		Description string `json:"description"`
	}

	jsonParams := map[string]any{}
	formParams := []FormParam{}
	queryParams := []*postman.QueryParam{}
	pathVariables := []*postman.Variable{}

	walkStructFields(typ, func(field reflect.StructField) {
		fieldName := field.Name
		jsonTag := field.Tag.Get("json")
		formTag := field.Tag.Get("form")
		formFileTag := field.Tag.Get("formFile")
		queryTag := field.Tag.Get("query")
		paramTag := field.Tag.Get("param")
		description := field.Tag.Get("description")
		example := field.Tag.Get("example")

		jsonKey := jsonTag
		if jsonKey == "" || jsonKey == "-" {
			jsonKey = fieldName
		}
		formKey := formTag
		if formKey == "" || formKey == "-" {
			formKey = fieldName
		}
		formFileKey := formFileTag
		if formFileKey == "" || formFileKey == "-" {
			formFileKey = fieldName
		}
		queryKey := queryTag
		if queryKey == "" || queryKey == "-" {
			queryKey = fieldName
		}
		paramKey := paramTag
		if paramKey == "" || paramKey == "-" {
			paramKey = fieldName
		}

		var placeholderValue any = example
		if placeholderValue == "" || placeholderValue == "-" {
			if defaultValue, ok := p.placeholderDefaults[jsonKey]; ok {
				placeholderValue = defaultValue
			} else if defaultValue, ok := p.placeholderDefaults[formKey]; ok {
				placeholderValue = defaultValue
			} else if defaultValue, ok := p.placeholderDefaults[formFileKey]; ok {
				placeholderValue = defaultValue
			} else if defaultValue, ok := p.placeholderDefaults[queryKey]; ok {
				placeholderValue = defaultValue
			} else if defaultValue, ok := p.placeholderDefaults[paramKey]; ok {
				placeholderValue = defaultValue
			}
		}

		if jsonTag != "" && jsonTag != "-" {
			value := placeholderValue
			if placeholderValue == "" || placeholderValue == "-" {
				value = p.TypeZeroValue(field.Type, false)
			}
			jsonParams[jsonKey] = value
		}

		if placeholderValue == "" || placeholderValue == "-" {
			placeholderValue = p.TypeZeroValue(field.Type, true)
		}

		if formTag != "" && formTag != "-" {
			formParams = append(formParams, FormParam{
				Key:         formKey,
				Value:       fmt.Sprint(placeholderValue),
				Type:        "text",
				Description: description,
			})
		}

		if formFileTag != "" && formFileTag != "-" {
			formParams = append(formParams, FormParam{
				Key:         formFileKey,
				Value:       fmt.Sprint(placeholderValue),
				Type:        "file",
				Description: description,
			})
		}

		if queryTag != "" && queryTag != "-" {
			queryParams = append(queryParams, &postman.QueryParam{
				Key:         queryKey,
				Value:       fmt.Sprint(placeholderValue),
				Description: &description,
			})
		}

		if paramTag != "" && paramTag != "-" {
			pathVariables = append(pathVariables, &postman.Variable{
				Key:   paramKey,
				Value: fmt.Sprint(placeholderValue),
				Type:  "string",
			})
		}
	})

	processedPath := path
	pathSegments := strings.Split(strings.Trim(processedPath, "/"), "/")
	urlVariables := []*postman.Variable{}

	for i, segment := range pathSegments {
		if strings.HasPrefix(segment, ":") {
			key := strings.TrimPrefix(segment, ":")
			// pathSegments[i] = "{{" + key + "}}"
			pathSegments[i] = ":" + key

			defaultValue := ":" + key
			description := ""
			for _, pv := range pathVariables {
				if pv.Key == key {
					defaultValue = pv.Value
					description = pv.Description
					break
				}
			}

			urlVariables = append(urlVariables, &postman.Variable{
				Key:         key,
				Value:       defaultValue,
				Type:        "string",
				Description: description,
			})
		}
	}
	processedPath = "/" + strings.Join(pathSegments, "/")

	request := &postman.Request{
		URL: &postman.URL{
			Raw:       "{{base_url}}" + processedPath,
			Host:      []string{"{{base_url}}"},
			Path:      pathSegments,
			Query:     queryParams,
			Variables: urlVariables,
		},
		Method: postman.Method(method),
		Header: []*postman.Header{},
		Body:   &postman.Body{},
	}

	if len(formParams) > 0 {
		request.Body.Mode = "formdata"
		request.Body.FormData = formParams
		request.Header = append(request.Header, &postman.Header{Key: "Content-Type", Value: "multipart/form-data"})
	} else if len(jsonParams) > 0 {
		bodyBytes, err := json.MarshalIndent(jsonParams, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal json body: %w", err)
		}
		request.Body.Mode = "raw"
		request.Body.Raw = string(bodyBytes)
		request.Body.Options = &postman.BodyOptions{
			Raw: postman.BodyOptionsRaw{Language: "json"},
		}
		request.Header = append(request.Header, &postman.Header{Key: "Content-Type", Value: "application/json"})
	}

	item := postman.CreateItem(postman.Item{
		Name:      pathSegments[len(pathSegments)-1],
		Request:   request,
		Responses: []*postman.Response{},
	})

	folderSegments := pathSegments[:len(pathSegments)-1]
	currentSlicePtr := &p.collection.Items

	for _, segment := range folderSegments {
		var foundFolder *postman.Items

		for _, existingItem := range *currentSlicePtr {
			if existingItem.Name == segment && existingItem.Request == nil {
				foundFolder = existingItem
				break
			}
		}

		if foundFolder == nil {
			newFolder := &postman.Items{
				Name:  segment,
				Items: make([]*postman.Items, 0),
			}
			*currentSlicePtr = append(*currentSlicePtr, newFolder)
			foundFolder = newFolder
		} else {
			if foundFolder.Items == nil {
				foundFolder.Items = make([]*postman.Items, 0)
			}
		}

		currentSlicePtr = &foundFolder.Items
	}

	*currentSlicePtr = append(*currentSlicePtr, item)

	return nil
}

func (p *PostmanGen) WriteToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	err = p.collection.Write(file, postman.V210)
	if err != nil {
		return err
	}
	return nil
}

func (p *PostmanGen) Write(w io.Writer) error {
	return p.collection.Write(w, postman.V210)
}

func (p *PostmanGen) TypeZeroValue(t reflect.Type, preferString bool) any {
	if t.Kind() == reflect.Ptr {
		return p.TypeZeroValue(t.Elem(), preferString)
	}
	if t.Kind() == reflect.Struct && preferString {
		zero := reflect.Zero(t)
		jsonBytes, err := json.Marshal(zero.Interface())
		if err != nil {
			return nil
		}
		return string(jsonBytes)
	}
	if t.Kind() == reflect.Slice {
		return []any{p.TypeZeroValue(t.Elem(), preferString)}
	}

	zero := reflect.Zero(t)
	return zero.Interface()
}
