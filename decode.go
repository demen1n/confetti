package confetti

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Decode populates v from an already-parsed *ConfigurationUnit.
// v must be a non-nil pointer to a struct.
func Decode(cfg *ConfigurationUnit, v any) error {
	if cfg == nil {
		return fmt.Errorf("confetti: Decode called with nil *ConfigurationUnit")
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("confetti: Decode requires a non-nil pointer, got %T", v)
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("confetti: Decode requires a pointer to a struct, got pointer to %s", rv.Kind())
	}
	return decodeStruct(cfg.Directives, rv)
}

// Unmarshal parses input with no extensions enabled, then calls Decode.
func Unmarshal(input string, v any) error {
	return UnmarshalWithOptions(input, v, Options{})
}

// UnmarshalWithOptions parses input with the given extension options, then calls Decode.
func UnmarshalWithOptions(input string, v any, opts Options) error {
	cfg, err := ParseWithOptions(input, opts)
	if err != nil {
		return err
	}
	return Decode(cfg, v)
}

// fieldInfo holds metadata about a struct field relevant to decoding.
type fieldInfo struct {
	index int
}

// structMeta is the result of inspecting a struct type.
type structMeta struct {
	byName      map[string]fieldInfo // confetti-name → field index
	argFieldIdx int                  // index of the ",arg" field, -1 if none
}

// fieldMap inspects t (must be a struct Type) and returns structMeta.
func fieldMap(t reflect.Type) structMeta {
	meta := structMeta{
		byName:      make(map[string]fieldInfo),
		argFieldIdx: -1,
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("conf")
		if tag == "-" {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		isArg := opts == "arg"

		if isArg {
			meta.argFieldIdx = i
			continue
		}

		if name == "" {
			name = strings.ToLower(f.Name)
		}
		meta.byName[name] = fieldInfo{index: i}
	}
	return meta
}

// decodeStruct populates the struct value rv from the given directives.
func decodeStruct(directives []Directive, rv reflect.Value) error {
	t := rv.Type()
	meta := fieldMap(t)

	for _, dir := range directives {
		if len(dir.Arguments) == 0 {
			continue
		}
		key := dir.Arguments[0]
		extraArgs := dir.Arguments[1:]

		fi, ok := meta.byName[key]
		if !ok {
			// unknown directive — silently ignore
			continue
		}

		fv := rv.Field(fi.index)
		ft := t.Field(fi.index)

		if err := decodeField(fv, ft.Type, extraArgs, dir.Subdirectives); err != nil {
			return fmt.Errorf("confetti: field %q: %w", key, err)
		}
	}
	return nil
}

// decodeField sets field fv (of type fieldType) from extraArgs and subdirectives.
func decodeField(fv reflect.Value, fieldType reflect.Type, extraArgs []string, subdirs []Directive) error {
	switch fieldType.Kind() {
	case reflect.Slice:
		elemType := fieldType.Elem()
		// []Struct or []*Struct — append a new element decoded from subdirectives
		if elemType.Kind() == reflect.Struct ||
			(elemType.Kind() == reflect.Pointer && elemType.Elem().Kind() == reflect.Struct) {
			return appendStructElem(fv, elemType, extraArgs, subdirs)
		}
		// slice of scalars ([]string, []int, []time.Duration, ...) — collect all extra args
		return setScalarSlice(fv, extraArgs)

	case reflect.Struct:
		return decodeBlockIntoStruct(fv, extraArgs, subdirs)

	case reflect.Pointer:
		if fieldType.Elem().Kind() == reflect.Struct {
			if fv.IsNil() {
				fv.Set(reflect.New(fieldType.Elem()))
			}
			return decodeBlockIntoStruct(fv.Elem(), extraArgs, subdirs)
		}
		return fmt.Errorf("unsupported pointer element type %s", fieldType.Elem().Kind())

	default:
		// scalar
		if len(extraArgs) == 0 {
			return fmt.Errorf("no value provided")
		}
		return setScalar(fv, extraArgs[0])
	}
}

// appendStructElem decodes a block directive into a new slice element and appends it.
func appendStructElem(fv reflect.Value, elemType reflect.Type, extraArgs []string, subdirs []Directive) error {
	isPtr := elemType.Kind() == reflect.Pointer
	var structType reflect.Type
	if isPtr {
		structType = elemType.Elem()
	} else {
		structType = elemType
	}
	if structType.Kind() != reflect.Struct {
		return fmt.Errorf("unsupported slice element type %s", elemType)
	}

	newElem := reflect.New(structType).Elem()
	if err := decodeBlockIntoStruct(newElem, extraArgs, subdirs); err != nil {
		return err
	}

	var toAppend reflect.Value
	if isPtr {
		ptr := reflect.New(structType)
		ptr.Elem().Set(newElem)
		toAppend = ptr
	} else {
		toAppend = newElem
	}
	fv.Set(reflect.Append(fv, toAppend))
	return nil
}

// decodeBlockIntoStruct decodes subdirectives into sv (a struct Value) and sets
// the ",arg" field (if any) from extraArgs.
func decodeBlockIntoStruct(sv reflect.Value, extraArgs []string, subdirs []Directive) error {
	meta := fieldMap(sv.Type())

	// set inline args
	if err := setArgField(sv, meta.argFieldIdx, extraArgs); err != nil {
		return err
	}

	// recurse into subdirectives
	return decodeStruct(subdirs, sv)
}

// setArgField populates the ",arg" field at argIdx in rv from args.
func setArgField(rv reflect.Value, argIdx int, args []string) error {
	if argIdx < 0 {
		return nil
	}
	fv := rv.Field(argIdx)
	switch fv.Kind() {
	case reflect.String:
		if len(args) > 0 {
			fv.SetString(args[0])
		}
	case reflect.Slice:
		if err := setScalarSlice(fv, args); err != nil {
			return fmt.Errorf(",arg field: %w", err)
		}
	default:
		return fmt.Errorf("unsupported ,arg field type %s", fv.Kind())
	}
	return nil
}

// setScalarSlice fills slice value fv with args, converting each element with setScalar.
func setScalarSlice(fv reflect.Value, args []string) error {
	sv := reflect.MakeSlice(fv.Type(), len(args), len(args))
	for i, a := range args {
		if err := setScalar(sv.Index(i), a); err != nil {
			return fmt.Errorf("element %d: %w", i, err)
		}
	}
	fv.Set(sv)
	return nil
}

var durationType = reflect.TypeOf(time.Duration(0))

// setScalar converts string s to the kind of rv and sets it.
func setScalar(rv reflect.Value, s string) error {
	if rv.Type() == durationType {
		d, err := time.ParseDuration(s)
		if err != nil {
			return fmt.Errorf("cannot parse %q as duration: %w", s, err)
		}
		rv.SetInt(int64(d))
		return nil
	}
	switch rv.Kind() {
	case reflect.String:
		rv.SetString(s)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("cannot parse %q as bool: %w", s, err)
		}
		rv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, rv.Type().Bits())
		if err != nil {
			return fmt.Errorf("cannot parse %q as %s: %w", s, rv.Kind(), err)
		}
		rv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, rv.Type().Bits())
		if err != nil {
			return fmt.Errorf("cannot parse %q as %s: %w", s, rv.Kind(), err)
		}
		rv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, rv.Type().Bits())
		if err != nil {
			return fmt.Errorf("cannot parse %q as %s: %w", s, rv.Kind(), err)
		}
		rv.SetFloat(f)
	default:
		return fmt.Errorf("unsupported type %s", rv.Kind())
	}
	return nil
}
