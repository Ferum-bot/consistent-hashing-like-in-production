package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const envTag = "env"

var durationType = reflect.TypeFor[time.Duration]()

func Load(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	if err := decoder.Decode(dst); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	return applyEnv(reflect.ValueOf(dst).Elem())
}

func applyEnv(target reflect.Value) error {
	for i := range target.NumField() {
		field := target.Field(i)
		name := target.Type().Field(i).Tag.Get(envTag)

		if name == "" && field.Kind() == reflect.Struct {
			if err := applyEnv(field); err != nil {
				return err
			}

			continue
		}

		if name == "" {
			continue
		}

		if !supported(field.Type()) {
			return fmt.Errorf("%s: unsupported type %s", name, field.Type())
		}

		raw, ok := os.LookupEnv(name)
		if !ok {
			continue
		}

		if err := set(field, raw); err != nil {
			return fmt.Errorf("%s=%q: %w", name, raw, err)
		}
	}

	return nil
}

func supported(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String, reflect.Int, reflect.Int64, reflect.Float64, reflect.Bool:
		return true
	case reflect.Slice:
		return t.Elem().Kind() == reflect.String
	default:
		return false
	}
}

func set(field reflect.Value, raw string) error {
	if field.Type() == durationType {
		value, err := time.ParseDuration(raw)
		if err != nil {
			return err
		}

		field.SetInt(int64(value))

		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Int, reflect.Int64:
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}

		field.SetInt(value)
	case reflect.Float64:
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}

		field.SetFloat(value)
	case reflect.Bool:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}

		field.SetBool(value)
	case reflect.Slice:
		field.Set(reflect.ValueOf(splitList(raw)))
	}

	return nil
}

func splitList(raw string) []string {
	var items []string

	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}

	return items
}
