package router

import (
	"github.com/alucardeht/may-la-mcp/internal/index"
	"github.com/alucardeht/may-la-mcp/internal/types"
)

func FromIndexedSymbol(indexed *index.IndexedSymbol) types.Symbol {
	return types.Symbol{
		Name:          indexed.Name,
		Kind:          indexed.Kind,
		File:          "",
		Line:          indexed.LineStart,
		LineEnd:       indexed.LineEnd,
		Column:        indexed.ColumnStart,
		ColumnEnd:     indexed.ColumnEnd,
		Signature:     indexed.Signature,
		Documentation: indexed.Documentation,
		IsExported:    indexed.IsExported,
	}
}

func FromSearchSymbol(searchSym types.Symbol) types.Symbol {
	return types.Symbol{
		Name:      searchSym.Name,
		Kind:      searchSym.Kind,
		File:      searchSym.File,
		Line:      searchSym.Line,
		Signature: searchSym.Signature,
	}
}

func FromSearchReference(searchRef types.Reference) types.Reference {
	return types.Reference{
		File:    searchRef.File,
		Line:    searchRef.Line,
		Column:  searchRef.Column,
		Context: searchRef.Context,
		Kind:    searchRef.Kind,
	}
}

func FromIndexedReference(indexed *index.SymbolReference) types.Reference {
	return types.Reference{
		File:    "",
		Line:    indexed.Line,
		Column:  indexed.Column,
		Context: indexed.Context,
		Kind:    indexed.Kind,
	}
}

func ToSearchSymbol(sym types.Symbol) types.Symbol {
	return types.Symbol{
		Name:      sym.Name,
		Kind:      sym.Kind,
		File:      sym.File,
		Line:      sym.Line,
		Signature: sym.Signature,
	}
}

func ToSearchReference(ref types.Reference) types.Reference {
	return types.Reference{
		File:    ref.File,
		Line:    ref.Line,
		Column:  ref.Column,
		Context: ref.Context,
		Kind:    ref.Kind,
	}
}

func SymbolsFromSearch(searchSymbols []types.Symbol) []types.Symbol {
	symbols := make([]types.Symbol, len(searchSymbols))
	for i, s := range searchSymbols {
		symbols[i] = FromSearchSymbol(s)
	}
	return symbols
}

func SymbolsFromIndexed(indexedSymbols []*index.IndexedSymbol) []types.Symbol {
	symbols := make([]types.Symbol, len(indexedSymbols))
	for i, s := range indexedSymbols {
		symbols[i] = FromIndexedSymbol(s)
	}
	return symbols
}

func ReferencesFromSearch(searchRefs []types.Reference) []types.Reference {
	refs := make([]types.Reference, len(searchRefs))
	for i, r := range searchRefs {
		refs[i] = FromSearchReference(r)
	}
	return refs
}

func ReferencesFromIndexed(indexedRefs []*index.SymbolReference) []types.Reference {
	refs := make([]types.Reference, len(indexedRefs))
	for i, r := range indexedRefs {
		refs[i] = FromIndexedReference(r)
	}
	return refs
}
