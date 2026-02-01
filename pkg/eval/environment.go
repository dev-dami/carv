package eval

type Environment struct {
	store     map[string]Object
	immutable map[string]bool
	outer     *Environment
}

func NewEnvironment() *Environment {
	return &Environment{
		store:     make(map[string]Object),
		immutable: make(map[string]bool),
		outer:     nil,
	}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

func (e *Environment) SetImmutable(name string, val Object) Object {
	e.store[name] = val
	e.immutable[name] = true
	return val
}

func (e *Environment) IsImmutable(name string) bool {
	if imm, ok := e.immutable[name]; ok {
		return imm
	}
	if e.outer != nil {
		return e.outer.IsImmutable(name)
	}
	return false
}

func (e *Environment) Update(name string, val Object) (Object, bool) {
	_, ok := e.store[name]
	if ok {
		e.store[name] = val
		return val, true
	}
	if e.outer != nil {
		return e.outer.Update(name, val)
	}
	return nil, false
}
