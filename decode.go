package confetti

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
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

// Unmarshal parses input then calls Decode. Convenience wrapper.
func Unmarshal(input string, v any) error {
	p, err := NewParser(input)
	if err != nil {
		return fmt.Errorf("confetti: Unmarshal parse init: %w", err)
	}
	cfg, err := p.Parse()
	if err != nil {
		return fmt.Errorf("confetti: Unmarshal parse: %w", err)
	}
	return Decode(cfg, v)
}

// fieldInfo holds metadata about a struct field relevant to decoding.
type fieldInfo struct {
	index    int
	argField bool // true if this field bears the ",arg" tag
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

		if err := decodeField(fv, ft.Type, extraArgs, dir.Subdirectives, meta.argFieldIdx, rv); err != nil {
			return fmt.Errorf("confetti: field %q: %w", key, err)
		}
	}
	return nil
}

// decodeField sets field fv (of type fieldType) from extraArgs and subdirectives.
// argFieldIdx and parentRV are used for the parent struct's ",arg" field when decoding
// block directives whose args target the parent — but at this call level we're targeting fv
// itself, so argFieldIdx / parentRV are not used here. They live in decodeStruct above.
func decodeField(fv reflect.Value, fieldType reflect.Type, extraArgs []string, subdirs []Directive, _ int, _ reflect.Value) error {
	switch fieldType.Kind() {
	case reflect.Slice:
		elemType := fieldType.Elem()
		// []string  — collect all extra args
		if elemType.Kind() == reflect.String {
			sv := reflect.MakeSlice(fieldType, len(extraArgs), len(extraArgs))
			for i, a := range extraArgs {
				sv.Index(i).SetString(a)
			}
			fv.Set(sv)
			return nil
		}
		// []Struct or []*Struct — append a new element decoded from subdirectives
		return appendStructElem(fv, fieldType, elemType, extraArgs, subdirs)

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
func appendStructElem(fv reflect.Value, sliceType, elemType reflect.Type, extraArgs []string, subdirs []Directive) error {
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
		if fv.Type().Elem().Kind() == reflect.String {
			sv := reflect.MakeSlice(fv.Type(), len(args), len(args))
			for i, a := range args {
				sv.Index(i).SetString(a)
			}
			fv.Set(sv)
		} else {
			return fmt.Errorf("unsupported ,arg slice element type %s", fv.Type().Elem().Kind())
		}
	default:
		return fmt.Errorf("unsupported ,arg field type %s", fv.Kind())
	}
	return nil
}

// setScalar converts string s to the kind of rv and sets it.
func setScalar(rv reflect.Value, s string) error {
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
