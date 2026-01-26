package lsp

import "time"

type LSPState string

const (
	StateStopped      LSPState = "stopped"
	StateStarting     LSPState = "starting"
	StateInitializing LSPState = "initializing"
	StateReady        LSPState = "ready"
	StateIdle         LSPState = "idle"
	StateError        LSPState = "error"
)

type Language string

const (
	LangGo         Language = "go"
	LangTypeScript Language = "typescript"
	LangJavaScript Language = "javascript"
	LangPython     Language = "python"
	LangRust       Language = "rust"
	LangCpp        Language = "cpp"
	LangC          Language = "c"
	LangJava       Language = "java"
)

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           SymbolKind       `json:"kind"`
	Tags           []SymbolTag      `json:"tags,omitempty"`
	Deprecated     bool             `json:"deprecated,omitempty"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

type SymbolInformation struct {
	Name          string     `json:"name"`
	Kind          SymbolKind `json:"kind"`
	Tags          []SymbolTag `json:"tags,omitempty"`
	Deprecated    bool       `json:"deprecated,omitempty"`
	Location      Location   `json:"location"`
	ContainerName string     `json:"containerName,omitempty"`
}

type SymbolKind int

const (
	SymbolKindFile          SymbolKind = 1
	SymbolKindModule        SymbolKind = 2
	SymbolKindNamespace     SymbolKind = 3
	SymbolKindPackage       SymbolKind = 4
	SymbolKindClass         SymbolKind = 5
	SymbolKindMethod        SymbolKind = 6
	SymbolKindProperty      SymbolKind = 7
	SymbolKindField         SymbolKind = 8
	SymbolKindConstructor   SymbolKind = 9
	SymbolKindEnum          SymbolKind = 10
	SymbolKindInterface     SymbolKind = 11
	SymbolKindFunction      SymbolKind = 12
	SymbolKindVariable      SymbolKind = 13
	SymbolKindConstant      SymbolKind = 14
	SymbolKindString        SymbolKind = 15
	SymbolKindNumber        SymbolKind = 16
	SymbolKindBoolean       SymbolKind = 17
	SymbolKindArray         SymbolKind = 18
	SymbolKindObject        SymbolKind = 19
	SymbolKindKey           SymbolKind = 20
	SymbolKindNull          SymbolKind = 21
	SymbolKindEnumMember    SymbolKind = 22
	SymbolKindStruct        SymbolKind = 23
	SymbolKindEvent         SymbolKind = 24
	SymbolKindOperator      SymbolKind = 25
	SymbolKindTypeParameter SymbolKind = 26
)

func (k SymbolKind) String() string {
	names := map[SymbolKind]string{
		SymbolKindFile:          "file",
		SymbolKindModule:        "module",
		SymbolKindNamespace:     "namespace",
		SymbolKindPackage:       "package",
		SymbolKindClass:         "class",
		SymbolKindMethod:        "method",
		SymbolKindProperty:      "property",
		SymbolKindField:         "field",
		SymbolKindConstructor:   "constructor",
		SymbolKindEnum:          "enum",
		SymbolKindInterface:     "interface",
		SymbolKindFunction:      "function",
		SymbolKindVariable:      "variable",
		SymbolKindConstant:      "constant",
		SymbolKindString:        "string",
		SymbolKindNumber:        "number",
		SymbolKindBoolean:       "boolean",
		SymbolKindArray:         "array",
		SymbolKindObject:        "object",
		SymbolKindKey:           "key",
		SymbolKindNull:          "null",
		SymbolKindEnumMember:    "enumMember",
		SymbolKindStruct:        "struct",
		SymbolKindEvent:         "event",
		SymbolKindOperator:      "operator",
		SymbolKindTypeParameter: "typeParameter",
	}
	if name, ok := names[k]; ok {
		return name
	}
	return "unknown"
}

type SymbolTag int

const (
	SymbolTagDeprecated SymbolTag = 1
)

type LSPStats struct {
	Language     Language      `json:"language"`
	State        LSPState      `json:"state"`
	RequestCount int64         `json:"request_count"`
	ErrorCount   int64         `json:"error_count"`
	StartedAt    time.Time     `json:"started_at,omitempty"`
	LastRequest  time.Time     `json:"last_request,omitempty"`
	LastError    time.Time     `json:"last_error,omitempty"`
	LastErrorMsg string        `json:"last_error_msg,omitempty"`
	Uptime       time.Duration `json:"uptime,omitempty"`
}

type InitializeParams struct {
	ProcessID    int         `json:"processId"`
	RootURI      string      `json:"rootUri"`
	Capabilities interface{} `json:"capabilities"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

type ServerCapabilities struct {
	TextDocumentSync        interface{} `json:"textDocumentSync,omitempty"`
	DocumentSymbolProvider  interface{} `json:"documentSymbolProvider,omitempty"`
}

type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}
