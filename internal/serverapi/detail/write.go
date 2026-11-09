// Purpose: Serialize only public detail DTOs incrementally without a full JSON document buffer.
// Pattern: Private streaming writer over the explicit detail schema.
// Usage: The cache writes documentValue directly into its bounded memory or private spool.
// Related: internal/serverapi/detail/document.go, internal/serverapi/detail/write_string.go, internal/serverapi/detail/spool.go.
package detail

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/keonho-kim/meca/internal/commands/server"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type documentWriter struct {
	ctx     context.Context
	output  io.Writer
	newMask func() func(string, bool) string
	quoted  bytes.Buffer
	encoder *json.Encoder
}

func writeDocument(ctx context.Context, output io.Writer, original server.DetailOriginal, newMask func() func(string, bool) string) error {
	if newMask == nil {
		return faults.Errorf(diagcatalog.DetailMaskFactoryRequired)
	}
	value, err := documentValue(original)
	if err != nil {
		return err
	}
	buffer := bufio.NewWriterSize(output, 32<<10)
	w := documentWriter{ctx: ctx, output: buffer, newMask: newMask}
	if err := w.value(reflect.ValueOf(value)); err != nil {
		return err
	}
	return buffer.Flush()
}

func (w *documentWriter) literal(value string) error {
	if err := w.ctx.Err(); err != nil {
		return err
	}
	_, err := io.WriteString(w.output, value)
	return err
}

func (w *documentWriter) value(value reflect.Value) error {
	if err := w.ctx.Err(); err != nil {
		return err
	}
	if !value.IsValid() {
		return w.literal("null")
	}
	if value.Type() == reflect.TypeFor[json.RawMessage]() {
		return w.raw(value.Bytes())
	}
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return w.literal("null")
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.String:
		return w.text(value.String())
	case reflect.Struct:
		return w.object(value)
	case reflect.Slice, reflect.Array:
		if value.Kind() == reflect.Slice && value.IsNil() {
			return w.literal("null")
		}
		if err := w.literal("["); err != nil {
			return err
		}
		for i := 0; i < value.Len(); i++ {
			if i > 0 {
				if err := w.literal(","); err != nil {
					return err
				}
			}
			if err := w.value(value.Index(i)); err != nil {
				return err
			}
		}
		return w.literal("]")
	case reflect.Map:
		return w.mapping(value)
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		body, err := json.Marshal(value.Interface())
		if err != nil {
			return err
		}
		_, err = w.output.Write(body)
		return err
	default:
		return faults.Errorf(diagcatalog.DetailUnsupportedValue)
	}
}

func emptyValue(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return value.Len() == 0
	}
	return value.IsZero()
}

func (w *documentWriter) object(value reflect.Value) error {
	if err := w.literal("{"); err != nil {
		return err
	}
	comma := false
	shape := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := shape.Field(i)
		if field.PkgPath != "" {
			continue
		}
		options := strings.Split(field.Tag.Get("json"), ",")
		name := options[0]
		if name == "-" {
			continue
		}
		if name == "" {
			name = field.Name
		}
		item := value.Field(i)
		omit, quoted := false, false
		for _, option := range options[1:] {
			omit = omit || option == "omitempty"
			quoted = quoted || option == "string"
		}
		if omit && emptyValue(item) {
			continue
		}
		if comma {
			if err := w.literal(","); err != nil {
				return err
			}
		}
		comma = true
		// Owned schema keys are protocol identifiers; only values and untrusted
		// raw JSON keys are credential-bearing input.
		key, err := json.Marshal(name)
		if err != nil {
			return err
		}
		if _, err := w.output.Write(key); err != nil {
			return err
		}
		if err := w.literal(":"); err != nil {
			return err
		}
		if quoted {
			var text string
			switch item.Kind() {
			case reflect.Int, reflect.Int64:
				text = strconv.FormatInt(item.Int(), 10)
			case reflect.Uint64:
				text = strconv.FormatUint(item.Uint(), 10)
			default:
				return faults.Errorf(diagcatalog.DetailUnsupportedValue)
			}
			if err := w.text(text); err != nil {
				return err
			}
		} else if err := w.value(item); err != nil {
			return err
		}
	}
	return w.literal("}")
}

func (w *documentWriter) mapping(value reflect.Value) error {
	if value.IsNil() {
		return w.literal("null")
	}
	if value.Type().Key().Kind() != reflect.String {
		return faults.Errorf(diagcatalog.DetailUnsupportedValue)
	}
	keys := value.MapKeys()
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	if err := w.literal("{"); err != nil {
		return err
	}
	for i, key := range keys {
		if i > 0 {
			if err := w.literal(","); err != nil {
				return err
			}
		}
		if err := w.text(key.String()); err != nil {
			return err
		}
		if err := w.literal(":"); err != nil {
			return err
		}
		if err := w.value(value.MapIndex(key)); err != nil {
			return err
		}
	}
	return w.literal("}")
}

func writePlainText(ctx context.Context, output io.Writer, text string, newMask func() func(string, bool) string) error {
	if newMask == nil {
		return faults.Errorf(diagcatalog.DetailMaskFactoryRequired)
	}
	mask := newMask()
	for len(text) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		n := min(32<<10, len(text))
		if _, err := io.WriteString(output, mask(text[:n], false)); err != nil {
			return err
		}
		text = text[n:]
	}
	_, err := io.WriteString(output, mask("", true))
	return err
}
