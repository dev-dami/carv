package types

import "github.com/dev-dami/carv/pkg/ast"

func (c *Checker) checkClassStatement(s *ast.ClassStatement) {
	fields := make(map[string]Type)
	for _, f := range s.Fields {
		var ft Type
		if f.Type != nil {
			ft = c.resolveTypeExpr(f.Type)
		} else {
			ft = Any
		}
		fields[f.Name.Value] = ft
	}

	classType := &ClassType{Name: s.Name.Value, Fields: fields}
	c.scope.Define(s.Name.Value, classType)

	for _, method := range s.Methods {
		prevScope := c.scope
		prevOwnership := c.pushOwnership()
		prevBorrows := c.pushBorrows()
		c.scope = NewScope(prevScope)

		switch method.Receiver {
		case ast.RecvRef:
			c.scope.Define("self", &RefType{Inner: classType, Mutable: false})
		case ast.RecvMutRef:
			c.scope.Define("self", &RefType{Inner: classType, Mutable: true})
		case ast.RecvValue:
			c.scope.Define("self", classType)
		}

		paramTypes := c.resolveParameterTypes(method.Parameters)
		for i, p := range method.Parameters {
			c.scope.Define(p.Name.Value, paramTypes[i])
		}

		if method.Body != nil {
			c.checkBlockStatement(method.Body)
		}

		c.scope = prevScope
		c.popOwnership(prevOwnership)
		c.popBorrows(prevBorrows)
	}
}

func (c *Checker) checkInterfaceStatement(s *ast.InterfaceStatement) {
	methods := make(map[string]*FunctionType)
	receivers := make(map[string]ast.ReceiverKind)

	for _, sig := range s.Methods {
		paramTypes := c.resolveParameterTypes(sig.Parameters)

		var retType Type = Void
		if sig.ReturnType != nil {
			retType = c.resolveTypeExpr(sig.ReturnType)
		}

		methods[sig.Name.Value] = &FunctionType{Params: paramTypes, Return: retType}
		receivers[sig.Name.Value] = sig.Receiver
	}

	ifaceType := &InterfaceType{Name: s.Name.Value, Methods: methods}
	c.scope.Define(s.Name.Value, ifaceType)
	c.ifaceReceivers[s.Name.Value] = receivers
}

func (c *Checker) checkImplStatement(s *ast.ImplStatement) {
	ifaceType, ok := c.scope.Lookup(s.Interface.Value)
	if !ok {
		line, col := s.Interface.Pos()
		c.error(line, col, "undefined interface: %s", s.Interface.Value)
		return
	}
	iface, ok := ifaceType.(*InterfaceType)
	if !ok {
		line, col := s.Interface.Pos()
		c.error(line, col, "%s is not an interface", s.Interface.Value)
		return
	}

	classType, ok := c.scope.Lookup(s.Type.Value)
	if !ok {
		line, col := s.Type.Pos()
		c.error(line, col, "undefined type: %s", s.Type.Value)
		return
	}

	implMethods := make(map[string]*FunctionType)
	implReceivers := make(map[string]ast.ReceiverKind)
	implDecls := make(map[string]*ast.MethodDecl)
	for _, method := range s.Methods {
		paramTypes := c.resolveParameterTypes(method.Parameters)

		var retType Type = Void
		if method.ReturnType != nil {
			retType = c.resolveTypeExpr(method.ReturnType)
		}

		implMethods[method.Name.Value] = &FunctionType{Params: paramTypes, Return: retType}
		implReceivers[method.Name.Value] = method.Receiver
		implDecls[method.Name.Value] = method

		prevScope := c.scope
		prevOwnership := c.pushOwnership()
		prevBorrows := c.pushBorrows()
		c.scope = NewScope(prevScope)

		switch method.Receiver {
		case ast.RecvRef:
			c.scope.Define("self", &RefType{Inner: classType, Mutable: false})
		case ast.RecvMutRef:
			c.scope.Define("self", &RefType{Inner: classType, Mutable: true})
		case ast.RecvValue:
			c.scope.Define("self", classType)
		}

		for i, p := range method.Parameters {
			c.scope.Define(p.Name.Value, paramTypes[i])
		}

		if method.Body != nil {
			c.checkBlockStatement(method.Body)
		}

		c.scope = prevScope
		c.popOwnership(prevOwnership)
		c.popBorrows(prevBorrows)
	}

	for name, ifaceMethod := range iface.Methods {
		implMethod, exists := implMethods[name]
		if !exists {
			line, col := s.Pos()
			c.error(line, col, "type %s does not implement %s: missing method %s",
				s.Type.Value, s.Interface.Value, name)
			continue
		}
		if len(implMethod.Params) != len(ifaceMethod.Params) {
			line, col := s.Pos()
			c.error(line, col, "method %s has wrong number of parameters", name)
			continue
		}
		for i, p := range ifaceMethod.Params {
			if !c.isAssignable(p, implMethod.Params[i]) {
				line, col := s.Pos()
				c.error(line, col, "method %s parameter %d: expected %s, got %s",
					name, i+1, p.String(), implMethod.Params[i].String())
			}
		}
		if !c.isAssignable(ifaceMethod.Return, implMethod.Return) {
			line, col := s.Pos()
			c.error(line, col, "method %s: return type mismatch: expected %s, got %s",
				name, ifaceMethod.Return.String(), implMethod.Return.String())
		}
		if ifaceReceiverMap, ok := c.ifaceReceivers[s.Interface.Value]; ok {
			if ifaceReceiver, ok := ifaceReceiverMap[name]; ok {
				if implReceiver, ok := implReceivers[name]; ok && ifaceReceiver != implReceiver {
					line, col := s.Pos()
					if decl, ok := implDecls[name]; ok {
						line, col = decl.Name.Pos()
					}
					c.warning(line, col, "receiver mismatch for method %s: interface expects %s, impl has %s",
						name, receiverKindName(ifaceReceiver), receiverKindName(implReceiver))
				}
			}
		}
	}

	if c.impls[s.Type.Value] == nil {
		c.impls[s.Type.Value] = make(map[string]bool)
	}
	c.impls[s.Type.Value][s.Interface.Value] = true
}

func (c *Checker) checkMemberExpressionForInterface(e *ast.MemberExpression, objType Type) Type {
	if ref, ok := objType.(*RefType); ok {
		if iface, ok := ref.Inner.(*InterfaceType); ok {
			if methodType, exists := iface.Methods[e.Member.Value]; exists {
				if ifaceReceiverMap, ok := c.ifaceReceivers[iface.Name]; ok {
					if recv, ok := ifaceReceiverMap[e.Member.Value]; ok {
						if recv == ast.RecvMutRef && !ref.Mutable {
							line, col := e.Member.Pos()
							c.error(line, col, "cannot call &mut self method '%s' through immutable interface reference", e.Member.Value)
						}
					}
				}
				return methodType
			}
			line, col := e.Member.Pos()
			c.error(line, col, "interface %s has no method %s", iface.Name, e.Member.Value)
			return Any
		}
	}
	return nil
}

func receiverKindName(kind ast.ReceiverKind) string {
	switch kind {
	case ast.RecvRef:
		return "&self"
	case ast.RecvMutRef:
		return "&mut self"
	case ast.RecvValue:
		return "self"
	case ast.RecvNone:
		return "none"
	default:
		return "unknown"
	}
}
