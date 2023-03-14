package codes

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/module"
	"github.com/aacfactory/gcg"
	"strings"
)

func MapTypeToFunctionElementCode(ctx context.Context, typ *module.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: map type to function document element code, failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
	}

	switch typ.Kind {
	case module.BasicKind:
		code, err = mapBasicTypeToFunctionElementCode(ctx, typ)
		break
	case module.BuiltinKind:

		break
	case module.IdentKind:

		break
	case module.InterfaceKind:

		break
	case module.StructKind:
		code, err = mapStructTypeToFunctionElementCode(ctx, typ)
		break
	case module.PointerKind:
		code, err = mapPointerTypeToFunctionElementCode(ctx, typ)
		break
	case module.ArrayKind:
		code, err = mapArrayTypeToFunctionElementCode(ctx, typ)
		break
	case module.MapKind:

		break
	case module.AnyKind:

	case module.ParadigmKind:
		// todo packed
		break
	case module.ParadigmElementKind:
		// todo packed
		break
	default:

	}
	return
}

func mapBasicTypeToFunctionElementCode(ctx context.Context, typ *module.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: map type to function document element code, failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
	}
	stmt := gcg.Statements()
	switch typ.Name {
	case "string":
		stmt.Token(fmt.Sprintf("documents.String()"))
		break
	case "bool":
		stmt.Token(fmt.Sprintf("documents.Bool()"))
		break
	case "int8", "int16", "int32":
		stmt.Token(fmt.Sprintf("documents.Int32()"))
		break
	case "int", "int64":
		stmt.Token(fmt.Sprintf("documents.Int64()"))
		break
	case "uint8", "byte":
		stmt.Token(fmt.Sprintf("documents.Uint8()"))
		break
	case "uint16", "uint32":
		stmt.Token(fmt.Sprintf("documents.Uint32()"))
		break
	case "uint", "uint64":
		stmt.Token(fmt.Sprintf("documents.Uint64()"))
		break
	case "float32":
		stmt.Token(fmt.Sprintf("documents.Float32()"))
		break
	case "float64":
		stmt.Token(fmt.Sprintf("documents.Float64()"))
		break
	case "complex64":
		stmt.Token(fmt.Sprintf("documents.Complex64()"))
		break
	case "complex128":
		stmt.Token(fmt.Sprintf("documents.Complex128()"))
		break
	default:
		err = errors.Warning("forg: unsupported basic type").WithMeta("name", typ.Name)
		return
	}
	code = stmt
	return
}

func mapPointerTypeToFunctionElementCode(ctx context.Context, typ *module.Type) (code gcg.Code, err error) {
	code, err = MapTypeToFunctionElementCode(ctx, typ.Elements[0])
	return
}

func mapStructTypeToFunctionElementCode(ctx context.Context, typ *module.Type) (code gcg.Code, err error) {
	if typ.ParadigmsPacked != nil {
		typ = typ.ParadigmsPacked
	}
	stmt := gcg.Statements()
	stmt = stmt.Token("documents.Struct(").Token(fmt.Sprintf("\"%s\",\"%s\"", typ.Path, typ.Name)).Symbol(")")
	title, hasTitle := typ.Annotations.Get("title")
	if hasTitle {
		stmt = stmt.Dot().Line().Token("SetTitle(").Token(fmt.Sprintf("\"%s\"", title)).Symbol(")")
	}
	description, hasDescription := typ.Annotations.Get("description")
	if hasDescription {
		stmt = stmt.Dot().Line().Token("SetDescription(").Token(fmt.Sprintf("\"%s\"", description)).Symbol(")")
	}
	_, hasDeprecated := typ.Annotations.Get("deprecated")
	if hasDeprecated {
		stmt = stmt.Dot().Line().Token("AsDeprecated()")
	}
	for _, field := range typ.Elements {
		name, hasName := field.Tags["json"]
		if !hasName {
			name = field.Name
		}
		if name == "-" {
			continue
		}
		elementCode, elementCodeErr := MapTypeToFunctionElementCode(ctx, field.Elements[0])
		if elementCodeErr != nil {
			err = errors.Warning("forg: map struct type to function element code failed").
				WithMeta("name", typ.Name).WithMeta("path", typ.Path).
				WithMeta("field", field.Name).
				WithCause(elementCodeErr)
			return
		}
		elementStmt := elementCode.(*gcg.Statement)
		fieldTitle, hasFieldTitle := field.Annotations.Get("title")
		if hasFieldTitle {
			elementStmt = elementStmt.Dot().Line().Token("SetTitle(").Token(fmt.Sprintf("\"%s\"", fieldTitle)).Symbol(")")
		}
		fieldDescription, hasFieldDescription := field.Annotations.Get("description")
		if hasFieldDescription {
			elementStmt = elementStmt.Dot().Line().Token("SetDescription(").Token(fmt.Sprintf("\"%s\"", fieldDescription)).Symbol(")")
		}
		_, hasFieldDeprecated := field.Annotations.Get("deprecated")
		if hasFieldDeprecated {
			elementStmt = elementStmt.Dot().Line().Token("AsDeprecated()")
		}
		// enum
		fieldEnum, hasFieldEnum := field.Annotations["enum"]
		if hasFieldEnum && fieldEnum != "" {
			fieldEnums := strings.Split(fieldEnum, ",")
			fieldEnumsCodeToken := ""
			for _, enumValue := range fieldEnums {
				fieldEnumsCodeToken = fieldEnumsCodeToken + `, "` + strings.TrimSpace(enumValue) + `"`
			}
			if fieldEnumsCodeToken != "" {
				fieldEnumsCodeToken = fieldEnumsCodeToken[2:]
				elementStmt = elementStmt.Dot().Line().Token("AddEnum").Symbol("(").Token(fieldEnumsCodeToken).Symbol(")")
			}
		}
		// validation
		fieldValidate, hasFieldValidate := field.Tags["validate"]
		if hasFieldValidate && fieldValidate != "" {
			fieldRequired := strings.Contains(fieldValidate, "required")
			if fieldRequired {
				elementStmt = elementStmt.Dot().Line().Token("AsRequired()")
			}
			fieldValidateMessage, hasFieldValidateMessage := field.Tags["validate-message"]
			if !hasFieldValidateMessage {
				fieldValidateMessage = field.Tags["message"]
			}
			fieldValidateMessageI18ns := make([]string, 0, 1)
			validateMessageI18n, hasValidateMessageI18n := field.Annotations.Get("validate-message-i18n")
			if hasValidateMessageI18n && validateMessageI18n != "" {
				reader := bufio.NewReader(bytes.NewReader([]byte(validateMessageI18n)))
				for {
					line, _, readErr := reader.ReadLine()
					if readErr != nil {
						break
					}
					idx := bytes.IndexByte(line, ':')
					if idx > 0 && idx < len(line) {
						fieldValidateMessageI18ns = append(fieldValidateMessageI18ns, strings.TrimSpace(string(line[0:idx])))
						fieldValidateMessageI18ns = append(fieldValidateMessageI18ns, strings.TrimSpace(string(line[idx+1:])))
					}
				}
			}
			fieldValidateMessageI18nsCodeToken := ""
			for _, fieldValidateMessageI18n := range fieldValidateMessageI18ns {
				fieldValidateMessageI18nsCodeToken = fieldValidateMessageI18nsCodeToken + `, "` + fieldValidateMessageI18n + `"`
			}
			if fieldValidateMessageI18nsCodeToken != "" {
				fieldValidateMessageI18nsCodeToken = fieldValidateMessageI18nsCodeToken[2:]
			}
			elementStmt = elementStmt.Dot().Line().Token("SetValidation(").
				Token("documents.NewElementValidation(").
				Token(fmt.Sprintf("\"%s\"", fieldValidateMessage)).
				Symbol(", ").
				Token(fieldValidateMessageI18nsCodeToken).
				Symbol(")").
				Symbol(")")
		}

		stmt = stmt.Dot().Line().
			Token("AddProperty(").Line().
			Token(fmt.Sprintf("\"%s\"", name)).Symbol(",").Line().
			Add(elementStmt).Symbol(",").Line().
			Symbol(")")
	}

	code = stmt
	return
}

func mapArrayTypeToFunctionElementCode(ctx context.Context, typ *module.Type) (code gcg.Code, err error) {
	element := typ.Elements[0]
	name, isBasic := element.Basic()
	if isBasic && (name == "byte" || name == "uint8") {
		stmt := gcg.Statements()
		stmt = stmt.Token(fmt.Sprintf("documents.Bytes()"))
		code = stmt
		return
	}
	elementCode, elementCodeErr := MapTypeToFunctionElementCode(ctx, element)
	if elementCodeErr != nil {
		err = errors.Warning("forg: map array type to function element code failed").
			WithMeta("name", typ.Name).WithMeta("path", typ.Path).
			WithCause(elementCodeErr)
		return
	}
	stmt := gcg.Statements()
	stmt = stmt.Token("documents.Array(").Add(elementCode).Symbol(")")
	title, hasTitle := typ.Annotations.Get("title")
	if hasTitle {
		stmt = stmt.Dot().Line().Token("SetTitle(").Token(fmt.Sprintf("\"%s\"", title)).Symbol(")")
	}
	description, hasDescription := typ.Annotations.Get("description")
	if hasDescription {
		stmt = stmt.Dot().Line().Token("SetDescription(").Token(fmt.Sprintf("\"%s\"", description)).Symbol(")")
	}
	code = stmt
	return
}
