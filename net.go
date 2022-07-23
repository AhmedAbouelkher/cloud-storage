package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
)

const (
	MaxUploadLimit = 1024 * 1024 * 1 // 1 MB
)

var (
	// ErrInvalidRequest is returned when the request is invalid.
	ErrInvalidRequest = errors.New("invalid request body")

	tlsConfig *tls.Config
)

var vdr *validator.Validate

type Payload map[string]any

func SendHttpJsonError(w http.ResponseWriter, status int, err error) error {
	return SendJson(w, status, Payload{
		"status": status,
		"error":  err.Error(),
	})
}

func SendValidationError(w http.ResponseWriter, err error, status int) error {
	s := status
	if s == 0 {
		s = http.StatusUnprocessableEntity
	}
	if err, ok := err.(ValidationError); ok {
		return SendJson(w, s, Payload{
			"status": s,
			"errors": err.Errors,
		})
	}

	return SendHttpJsonError(w, s, err)
}

func SendJson(w http.ResponseWriter, status int, p Payload) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(p)
}

func ParseBody(r *http.Request, v any) error {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil || len(b) == 0 {
		return ErrInvalidRequest
	}
	if err := json.Unmarshal(b, v); err != nil {
		return err
	}
	return nil
}

type ValidationError struct {
	Errors []string
}

// error method in validation error
func (e ValidationError) Error() string {
	return fmt.Sprintf("%v", e.Errors)
}

func ParseAndValidate(r *http.Request, v any) error {
	if err := ParseBody(r, v); err != nil {
		return err
	}
	if err := Validator().Struct(v); err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			return err
		}

		var ve ValidationError
		for _, err := range err.(validator.ValidationErrors) {
			ve.Errors = append(ve.Errors, err.Error())
		}
		return ve
	}
	return nil
}

func Validator() *validator.Validate {
	if vdr == nil {
		vdr = validator.New()
	}
	return vdr
}

func SetTLSConfigs(cfg *tls.Config) {
	tlsConfig = cfg
}

func AppUrl() string {
	dm := os.Getenv("APP_URL")
	p := "http://"
	if tlsConfig != nil {
		p = "https://"
	}
	return fmt.Sprintf("%s%s", p, dm)
}

func JoinUrl(path string) string {
	return fmt.Sprintf("%s%s", AppUrl(), path)
}

func GetAddr() string {
	prt := os.Getenv("PORT")
	return ":" + prt
}
