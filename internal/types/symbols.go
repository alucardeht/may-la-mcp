package types

type Symbol struct {
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	File          string `json:"file"`
	Line          int    `json:"line"`
	LineEnd       int    `json:"line_end,omitempty"`
	Column        int    `json:"column,omitempty"`
	ColumnEnd     int    `json:"column_end,omitempty"`
	Signature     string `json:"signature,omitempty"`
	Documentation string `json:"documentation,omitempty"`
	IsExported    bool   `json:"is_exported,omitempty"`
}

type Reference struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Context string `json:"context"`
	Kind    string `json:"kind"`
}
