package tools

func RegisterAll(r *Registry) {
	r.Register(&InfoTool{})
	r.Register(&ReadTool{})
	r.Register(&SearchTool{})
	r.Register(&FindTool{})
	r.Register(&SymbolsTool{})
}
