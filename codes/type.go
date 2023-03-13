package codes

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/module"
	"github.com/aacfactory/gcg"
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
	case module.StructFieldKind:

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

		break
	default:

	}
	return
}

func mapBasicTypeToFunctionElementCode(ctx context.Context, typ *module.Type) (code gcg.Code, err error) {
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

	return
}

func mapArrayTypeToFunctionElementCode(ctx context.Context, typ *module.Type) (code gcg.Code, err error) {
	element := typ.Elements[0]
	name, isBasic := element.Basic()
	if isBasic && (name == "byte" || name == "uint8") {
		stmt := gcg.Statements()
		stmt.Token(fmt.Sprintf("documents.Bytes()"))
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
		stmt = stmt.Token(".SetTitle(").Token(fmt.Sprintf("\"%s\"", title)).Symbol(")")
	}
	description, hasDescription := typ.Annotations.Get("description")
	if hasDescription {
		stmt = stmt.Token(".SetDescription(").Token(fmt.Sprintf("\"%s\"", description)).Symbol(")")
	}
	code = stmt
	return
}
