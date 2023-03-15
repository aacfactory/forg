package codes

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/forg/module"
	"github.com/aacfactory/gcg"
	"os"
	"path/filepath"
	"strings"
)

type ServiceFile struct {
	service *module.Service
}

func (s *ServiceFile) Name() (name string) {
	name = filepath.ToSlash(filepath.Join(s.service.Dir, "fns.go"))
	return
}

func (s *ServiceFile) Write(ctx context.Context) (err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: service write failed").
			WithMeta("service", s.service.Name).
			WithCause(ctx.Err())
		return
	}

	file := gcg.NewFileWithoutNote(s.service.Path[strings.LastIndex(s.service.Path, "/")+1:])

	file.FileComments("NOTE: this file was been automatically generated, DONT EDIT IT\n")

	packages, importsErr := s.importsCode(ctx)
	if importsErr != nil {
		err = errors.Warning("forg: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(importsErr)
		return
	}
	if packages != nil && len(packages) > 0 {
		for _, importer := range packages {
			file.AddImport(importer)
		}
	}

	names, namesErr := s.constFunctionNamesCode(ctx)
	if namesErr != nil {
		err = errors.Warning("forg: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(namesErr)
		return
	}
	file.AddCode(names)

	proxies, proxiesErr := s.proxyFunctionsCode(ctx)
	if proxiesErr != nil {
		err = errors.Warning("forg: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(proxiesErr)
		return
	}
	file.AddCode(proxies)

	service, serviceErr := s.serviceCode(ctx)
	if serviceErr != nil {
		err = errors.Warning("forg: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(serviceErr)
		return
	}
	file.AddCode(service)

	writer, openErr := os.OpenFile(s.Name(), os.O_TRUNC|os.O_RDWR|os.O_SYNC, 0600)
	if openErr != nil {
		err = errors.Warning("forg: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(openErr)
		return
	}
	renderErr := file.Render(writer)
	if renderErr != nil {
		_ = writer.Close()
		err = errors.Warning("forg: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(renderErr)
		return
	}
	_ = writer.Close()
	return
}

func (s *ServiceFile) importsCode(ctx context.Context) (packages []*gcg.Package, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: service write failed").
			WithMeta("service", s.service.Name).
			WithCause(ctx.Err())
		return
	}
	packages = make([]*gcg.Package, 0, 1)
	for _, i := range s.service.Imports {
		if i.Alias != "" {
			packages = append(packages, gcg.NewPackageWithAlias(i.Path, i.Alias))
		} else {
			packages = append(packages, gcg.NewPackage(i.Path))
		}
	}
	return
}

func (s *ServiceFile) constFunctionNamesCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: service write failed").
			WithMeta("service", s.service.Name).
			WithCause(ctx.Err())
		return
	}
	stmt := gcg.Constants()
	stmt.Add("_name", s.service.Name)
	for _, function := range s.service.Functions {
		stmt.Add(function.ConstIdent, function.Name())
	}
	code = stmt.Build()
	return
}

func (s *ServiceFile) proxyFunctionsCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: service write failed").
			WithMeta("service", s.service.Name).
			WithCause(ctx.Err())
		return
	}
	stmt := gcg.Statements()
	for _, function := range s.service.Functions {
		constIdent := function.ConstIdent
		proxyIdent := function.ProxyIdent
		proxy := gcg.Func()
		proxy.Name(proxyIdent)
		proxy.AddParam("ctx", gcg.QualifiedIdent(gcg.NewPackage("context"), "Context"))
		if function.Param != nil {
			var param gcg.Code = nil
			if s.service.Path == function.Param.Type.Path {
				param = gcg.Ident(function.Param.Type.Name)
			} else {
				pkg, hasPKG := s.service.Imports.Path(function.Param.Type.Path)
				if !hasPKG {
					err = errors.Warning("forg: make function proxy code failed").
						WithMeta("service", s.service.Name).
						WithMeta("function", function.Name()).
						WithCause(errors.Warning("import of param was not found").WithMeta("path", function.Param.Type.Path))
					return
				}
				if pkg.Alias == "" {
					param = gcg.QualifiedIdent(gcg.NewPackage(pkg.Path), function.Param.Type.Name)
				} else {
					param = gcg.QualifiedIdent(gcg.NewPackageWithAlias(pkg.Path, pkg.Alias), function.Param.Type.Name)
				}
			}
			proxy.AddParam("argument", param)
		}
		if function.Result != nil {
			var result gcg.Code = nil
			if s.service.Path == function.Result.Type.Path {
				result = gcg.Ident(function.Param.Type.Name)
			} else {
				pkg, hasPKG := s.service.Imports.Path(function.Result.Type.Path)
				if !hasPKG {
					err = errors.Warning("forg: make function proxy code failed").
						WithMeta("service", s.service.Name).
						WithMeta("function", function.Name()).
						WithCause(errors.Warning("import of result was not found").WithMeta("path", function.Result.Type.Path))
					return
				}
				if pkg.Alias == "" {
					result = gcg.QualifiedIdent(gcg.NewPackage(pkg.Path), function.Result.Type.Name)
				} else {
					result = gcg.QualifiedIdent(gcg.NewPackageWithAlias(pkg.Path, pkg.Alias), function.Result.Type.Name)
				}
			}
			proxy.AddResult("result", result)
		}
		proxy.AddResult("err", gcg.QualifiedIdent(gcg.NewPackage("github.com/aacfactory/errors"), "CodeError"))
		// body
		body := gcg.Statements()
		body.Tab().Ident("endpoint").Symbol(",").Space().Ident("hasEndpoint").Space().ColonEqual().Space().Token("service.GetEndpoint(ctx, _name)").Line()
		body.Tab().Token("if !hasEndpoint {").Line()
		body.Tab().Tab().Token(fmt.Sprintf("err = errors.Warning(\"%s: endpoint was not found\").WithMeta(\"name\", _name)", s.service.Name)).Line()
		body.Tab().Tab().Return().Line()
		body.Tab().Token("}").Line()
		if function.Param == nil {
			body.Tab().Token("argument := service.Empty").Line()
		}
		bodyArgumentCode := gcg.Statements().Token("service.NewArgument(argument)")
		bodyRequestCode := gcg.Statements().Token(fmt.Sprintf("service.NewRequest(ctx, _name, %s, ", constIdent)).Add(bodyArgumentCode).Symbol(")")
		if function.Result == nil {
			body.Tab().Token("_, err = endpoint.RequestSync(ctx, ").Add(bodyRequestCode).Symbol(")").Line()
		} else {
			body.Tab().Token("fr, requestErr := endpoint.RequestSync(ctx, ").Add(bodyRequestCode).Symbol(")").Line()
			body.Tab().Token("if requestErr != nil {").Line()
			body.Tab().Tab().Token("err = requestErr").Line()
			body.Tab().Tab().Return().Line()
			body.Tab().Token("}").Line()
			body.Tab().Token("if !fr.Exist() {").Line()
			body.Tab().Tab().Return().Line()
			body.Tab().Token("}").Line()
			body.Tab().Token("scanErr := fr.Scan(&result)").Line()
			body.Tab().Token("if scanErr != nil {").Line()
			body.Tab().Tab().Token(fmt.Sprintf("err = errors.Warning(\"%s: scan future result failed\")", s.service.Name)).Dot().Line()
			body.Tab().Tab().Tab().Token(fmt.Sprintf("WithMeta(\"service\", _name).WithMeta(\"fn\", %s)", constIdent)).Dot().Line()
			body.Tab().Tab().Tab().Token("WithCause(scanErr)").Line()
			body.Tab().Tab().Return().Line()
			body.Tab().Token("}").Line()
		}
		body.Tab().Return()
		proxy.Body(body)
		stmt = stmt.Add(proxy.Build()).Line()
	}
	code = stmt
	return
}

func (s *ServiceFile) serviceCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: service write failed").
			WithMeta("service", s.service.Name).
			WithCause(ctx.Err())
		return
	}
	stmt := gcg.Statements()
	instanceCode, instanceCodeErr := s.serviceInstanceCode(ctx)
	if instanceCodeErr != nil {
		err = instanceCodeErr
		return
	}
	stmt.Add(instanceCode).Line()

	typeCode, typeCodeErr := s.serviceTypeCode(ctx)
	if typeCodeErr != nil {
		err = typeCodeErr
		return
	}
	stmt.Add(typeCode).Line()

	handleFnCode, handleFnCodeErr := s.serviceHandleCode(ctx)
	if handleFnCodeErr != nil {
		err = handleFnCodeErr
		return
	}
	stmt.Add(handleFnCode).Line()

	docFnCode, docFnCodeErr := s.serviceDocumentCode(ctx)
	if docFnCodeErr != nil {
		err = docFnCodeErr
		return
	}
	stmt.Add(docFnCode).Line()

	code = stmt
	return
}

func (s *ServiceFile) serviceInstanceCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: service write failed").
			WithMeta("service", s.service.Name).
			WithCause(ctx.Err())
		return
	}
	instance := gcg.Func()
	instance.Name("Service")
	instance.AddResult("v", gcg.Token("service.Service"))
	body := gcg.Statements()
	body.Tab().Token("components := []service.Component{")
	if s.service.Components != nil && s.service.Components.Len() > 0 {
		path := fmt.Sprintf("%s/components", s.service.Path)
		for _, component := range s.service.Components {
			componentCode := gcg.QualifiedIdent(gcg.NewPackage(path), component.Indent)
			body.Line().Tab().Tab().Add(componentCode).Token("{}").Symbol(",")
		}
	}
	body.Symbol("}").Line()
	body.Tab().Token("v = &_service_{").Line()
	body.Tab().Tab().Token("Abstract: service.NewAbstract(").Line()
	body.Tab().Tab().Tab().Token("_name,").Line()
	if s.service.Internal {
		body.Tab().Tab().Tab().Token("true,").Line()
	} else {
		body.Tab().Tab().Tab().Token("false,").Line()
	}
	body.Tab().Tab().Tab().Token("components...,").Line()
	body.Tab().Tab().Symbol(")").Line()
	body.Tab().Return().Line()

	instance.Body(body)
	code = instance.Build()
	return
}

func (s *ServiceFile) serviceTypeCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: service write failed").
			WithMeta("service", s.service.Name).
			WithCause(ctx.Err())
		return
	}
	abstractFieldCode := gcg.StructField("")
	abstractFieldCode.Type(gcg.Token("service.Abstract"))
	serviceStructCode := gcg.Struct()
	serviceStructCode.AddField(abstractFieldCode)
	code = gcg.Type("_service_", serviceStructCode.Build())
	return
}

func (s *ServiceFile) serviceHandleCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: service write failed").
			WithMeta("service", s.service.Name).
			WithCause(ctx.Err())
		return
	}
	handleFnCode := gcg.Func()
	handleFnCode.Receiver("svc", gcg.Star().Ident("_service_"))
	handleFnCode.Name("Handle")
	handleFnCode.AddParam("ctx", gcg.QualifiedIdent(gcg.NewPackage("context"), "Context"))
	handleFnCode.AddParam("fn", gcg.String())
	handleFnCode.AddParam("argument", gcg.QualifiedIdent(gcg.NewPackage("service"), "Argument"))
	handleFnCode.AddResult("v", gcg.Token("interface{}"))
	handleFnCode.AddResult("err", gcg.QualifiedIdent(gcg.NewPackage("github.com/aacfactory/errors"), "CodeError"))

	body := gcg.Statements()
	if s.service.Functions != nil && s.service.Functions.Len() > 0 {
		fnSwitchCode := gcg.Switch()
		fnSwitchCode.Expression(gcg.Ident("fn"))
		for _, function := range s.service.Functions {
			functionCode := gcg.Statements()
			// internal
			if function.Internal() {
				functionCode.Token("// check internal").Line()
				functionCode.Token("if !service.CanAccessInternal(ctx) {").Line()
				functionCode.Tab().Token(fmt.Sprintf("err = errors.Warning(\"%s: %s cannot be accessed externally\")", s.service.Name, function.Name())).Line()
				functionCode.Tab().Return().Line()
				functionCode.Symbol("}").Line()
			}
			// authorization
			if function.Authorization() {
				functionCode.Token("// verify authorizations").Line()
				functionCode.Token("err = authorizations.Verify(ctx)", gcg.NewPackage("github.com/aacfactory/fns/endpoints/authorizations")).Line()
				functionCode.Token("if err != nil {").Line()
				functionCode.Tab().Return().Line()
				functionCode.Token("}").Line()
			}
			// permission
			if kind, has := function.Permission(); has {
				functionCode.Token("// verify permissions").Line()
				idx := strings.LastIndex(kind, "/")
				if idx < 0 || idx == len(kind)-1 {
					err = errors.Warning("forg: kind of function is invalid").
						WithMeta("service", s.service.Name).
						WithMeta("function", function.Name())
					return
				}
				kindIdent := kind[idx+1:]
				functionCode.Token(fmt.Sprintf("enforced, enforceErr := %s.EnforceRequest(ctx, _name, %s)", kindIdent, function.ConstIdent), gcg.NewPackage(kind)).Line()
				functionCode.Token("if enforceErr != nil {").Line()
				functionCode.Tab().Token(fmt.Sprintf("err = errors.Warning(\"%s: enforce request failed\").WithCause(enforceErr)", s.service.Name)).Line()
				functionCode.Tab().Return().Line()
				functionCode.Token("}").Line()
				functionCode.Token("if !enforced {").Line()
				functionCode.Tab().Token("err = errors.Forbidden(\"forbidden\")").Line()
				functionCode.Tab().Return().Line()
				functionCode.Token("}").Line()
			}
			// param
			if function.Param != nil {
				var param gcg.Code = nil
				if s.service.Path == function.Param.Type.Path {
					param = gcg.Ident(function.Param.Type.Name)
				} else {
					pkg, hasPKG := s.service.Imports.Path(function.Param.Type.Path)
					if !hasPKG {
						err = errors.Warning("forg: make service handle function code failed").
							WithMeta("service", s.service.Name).
							WithMeta("function", function.Name()).
							WithCause(errors.Warning("import of param was not found").WithMeta("path", function.Param.Type.Path))
						return
					}
					if pkg.Alias == "" {
						param = gcg.QualifiedIdent(gcg.NewPackage(pkg.Path), function.Param.Type.Name)
					} else {
						param = gcg.QualifiedIdent(gcg.NewPackageWithAlias(pkg.Path, pkg.Alias), function.Param.Type.Name)
					}
				}
				functionCode.Token("param := ").Add(param).Token("{}").Line()
				functionCode.Token("paramErr := argument.As(&param)")
				functionCode.Token("if paramErr != nil {").Line()
				functionCode.Tab().Token(fmt.Sprintf("err = errors.Warning(\"%s: decode request argument failed\").WithCause(paramErr)", s.service.Name)).Line()
				functionCode.Tab().Return().Line()
				functionCode.Token("}").Line()
				// param validation
				if title, has := function.Validation(); has {
					functionCode.Token(fmt.Sprintf("err = validators.Validate(param, \"%s\")", title), gcg.NewPackage("github.com/aacfactory/fns/service/validators")).Line()
					functionCode.Token("if err != nil {").Line()
					functionCode.Tab().Return().Line()
					functionCode.Token("}").Line()
				}
			}
			// timeout
			timeout, hasTimeout, timeoutErr := function.Timeout()
			if timeoutErr != nil {
				timeoutValue := function.Annotations["timeout"]
				err = errors.Warning("forg: make service handle function code failed").
					WithMeta("service", s.service.Name).
					WithMeta("function", function.Name()).
					WithCause(errors.Warning("value of @timeout is invalid").WithMeta("timeout", timeoutValue).WithCause(timeoutErr))
				return
			}
			if hasTimeout {
				functionCode.Token("var cancel context.CancelFunc = nil").Line()
				functionCode.Token(fmt.Sprintf("ctx, cancel = context.WithTimeout(ctx, time.Duration(%d))", int64(timeout))).Line()
			}
			// exec
			functionExecCode := gcg.Statements()
			// sql
			if db, has := function.SQL(); has {
				db = strings.TrimSpace(db)
				if db == "" {
					err = errors.Warning("forg: make service handle function code failed").
						WithMeta("service", s.service.Name).
						WithMeta("function", function.Name()).
						WithCause(errors.Warning("value of @sql is required"))
					return
				}
				functionExecCode.Token("// use sql database").Line()
				functionExecCode.Token(fmt.Sprintf("ctx = sql.WithOptions(ctx, sql.Database(\"%s\"))", db), gcg.NewPackage("github.com/aacfactory/fns-contrib/databases/sql")).Line()
			}
			// transactional
			if function.Transactional() {
				functionExecCode.Token("// sql begin transaction").Line()
				functionExecCode.Token("beginTransactionErr := sql.BeginTransaction(ctx)", gcg.NewPackage("github.com/aacfactory/fns-contrib/databases/sql")).Line()
				functionExecCode.Token("if beginTransactionErr != nil {").Line()
				functionExecCode.Tab().Token(fmt.Sprintf("err = errors.Warning(\"%s: begin sql transaction failed\").WithCause(beginTransactionErr)", s.service.Name)).Line()
				functionExecCode.Tab().Return().Line()
				functionExecCode.Token("}").Line()
			}
			// handle
			functionExecCode.Token("// execute function").Line()
			if function.Param != nil && function.Result != nil {
				functionExecCode.Token(fmt.Sprintf("v, err = %s(ctx, param)", function.Ident)).Line()
			} else if function.Param == nil && function.Result != nil {
				functionExecCode.Token(fmt.Sprintf("v, err = %s(ctx)", function.Ident)).Line()
			} else if function.Param != nil && function.Result == nil {
				functionExecCode.Token(fmt.Sprintf("err = %s(ctx, param)", function.Ident)).Line()
			} else if function.Param == nil && function.Result == nil {
				functionExecCode.Token(fmt.Sprintf("err = %s(ctx)", function.Ident)).Line()
			}
			if function.Transactional() {
				functionExecCode.Token("// sql commit transaction").Line()
				functionExecCode.Token("if err == nil {").Line()
				functionExecCode.Tab().Token("commitTransactionErr := sql.CommitTransaction(ctx)", gcg.NewPackage("github.com/aacfactory/fns-contrib/databases/sql")).Line()
				functionExecCode.Tab().Token("if commitTransactionErr == nil {").Line()
				functionExecCode.Tab().Tab().Token("_ = sql.RollbackTransaction(ctx)", gcg.NewPackage("github.com/aacfactory/fns-contrib/databases/sql")).Line()
				functionExecCode.Tab().Tab().Token(fmt.Sprintf("err = errors.ServiceError(\"%s: commit sql transaction failed\").WithCause(commitTransactionErr)", s.service.Name)).Line()
				functionExecCode.Tab().Tab().Return().Line()
				functionExecCode.Tab().Token("}").Line()
				functionExecCode.Token("}").Line()
			}

			// barrier
			if function.Barrier() {
				functionCode.Token("// barrier").Line()
				functionCode.Token(fmt.Sprintf("v, err = svc.Barrier(ctx, %s, argument, func() (v interface{}, err errors.CodeError) {", function.ConstIdent)).Line()
				functionCode.Add(functionExecCode)
				functionCode.Tab().Return().Line()
				functionCode.Token("})").Line()
			} else {
				functionCode.Add(functionExecCode)
			}
			if hasTimeout {
				functionCode.Token("cancel()")
			}
			functionCode.Return().Line()
			fnSwitchCode.Case(gcg.Ident(function.ConstIdent), functionCode)
		}
		notFoundCode := gcg.Statements()
		notFoundCode.Token(fmt.Sprintf("err = errors.Warning(\"%s: fn was not found\").WithMeta(\"service\", _name).WithMeta(\"fn\", fn)", s.service.Name)).Line()
		notFoundCode.Return().Line()
		fnSwitchCode.Default(notFoundCode)
		body.Add(fnSwitchCode.Build()).Line()
	}
	body.Return().Line()
	handleFnCode.Body(body)

	code = handleFnCode.Build()
	return
}

func (s *ServiceFile) serviceDocumentCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("forg: service write failed").
			WithMeta("service", s.service.Name).
			WithCause(ctx.Err())
		return
	}
	docFnCode := gcg.Func()
	docFnCode.Receiver("svc", gcg.Star().Ident("_service_"))
	docFnCode.Name("Document")
	docFnCode.AddResult("doc", gcg.QualifiedIdent(gcg.NewPackage("github.com/aacfactory/fns/service"), "Document"))
	body := gcg.Statements()
	if !s.service.Internal {
		fnCodes := make([]gcg.Code, 0, 1)
		for _, function := range s.service.Functions {
			if function.Internal() {
				continue
			}
			fnCode := gcg.Statements()
			fnCode.Token("// ").Token(function.Name()).Line()
			fnCode.Token("document.AddFn(").Line()
			fnCode.Tab().Token(fmt.Sprintf("\"%s\", \"%s\", \"%s\",%v, %v,", function.Name(), function.Title(), function.Description(), function.Authorization(), function.Deprecated())).Line()
			if function.Param != nil {
				paramCode, paramCodeErr := MapTypeToFunctionElementCode(ctx, function.Param.Type)
				if paramCodeErr != nil {
					err = errors.Warning("forg: make service document code failed").
						WithMeta("service", s.service.Name).
						WithMeta("function", function.Name()).
						WithCause(paramCodeErr)
					return
				}
				fnCode.Add(paramCode).Symbol(",").Line()
			} else {
				fnCode.Tab().Token("nil").Symbol(",").Line()
			}
			if function.Result != nil {
				resultCode, resultCodeErr := MapTypeToFunctionElementCode(ctx, function.Result.Type)
				if resultCodeErr != nil {
					err = errors.Warning("forg: make service document code failed").
						WithMeta("service", s.service.Name).
						WithMeta("function", function.Name()).
						WithCause(resultCodeErr)
					return
				}
				fnCode.Add(resultCode).Symbol(",").Line()
			} else {
				fnCode.Tab().Token("nil").Symbol(",").Line()
			}
			fnCode.Token(")").Line()
			fnCodes = append(fnCodes, fnCode)
		}
		if len(fnCodes) > 0 {
			body.Token(fmt.Sprintf("document := documents.NewService(_name, \"%s\")", s.service.Description), gcg.NewPackage("github.com/aacfactory/fns/service/documents")).Line()
			for _, fnCode := range fnCodes {
				body.Add(fnCode).Line()
			}
			body.Token("doc = document").Line()
		}
	}
	body.Return().Line()
	docFnCode.Body(body)
	code = docFnCode.Build()
	return
}
