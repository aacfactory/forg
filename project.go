package forg

type Project interface {
	Dependencies() (dependencies map[string]Dependency)
	Services() (services map[string]Service)
	Service(name string) (service Service, has bool)
	AddService() (process Process, err error)
}

type Service interface {
	Functions() (functions map[string]Function)
	Function(name string) (function Function, has bool)
	AddFunction() (process Process, err error)
}

type Function interface {
}

type Dependency interface {
}
