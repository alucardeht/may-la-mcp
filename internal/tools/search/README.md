# Search Tools

Conjunto de ferramentas para busca e navegação de código otimizado para performance.

## Ferramentas Disponíveis

### 1. Search Tool (`search`)

Busca por padrão em arquivos com suporte a regex e contexto.

**Parâmetros:**
- `pattern` (string, obrigatório): Padrão de busca ou regex
- `path` (string, obrigatório): Caminho raiz para buscar
- `recursive` (boolean, opcional): Buscar recursivamente em subdiretórios (padrão: true)
- `case_sensitive` (boolean, opcional): Busca sensível a maiúsculas (padrão: false)
- `regex` (boolean, opcional): Tratar padrão como regex (padrão: false)
- `context_lines` (integer, opcional): Linhas de contexto antes/depois do match (padrão: 0)
- `max_results` (integer, opcional): Máximo de resultados (padrão: 1000)

**Resposta:**
- `matches`: Array de matches com file, line, column, content, context
- `count`: Número total de matches
- `path`: Caminho raiz da busca

**Implementação:**
- Tenta usar `ripgrep` (rg) se disponível para máxima performance
- Fallback para implementação Go com suporte a regex e busca de texto simples

### 2. Find Tool (`find`)

Busca arquivos por nome usando glob patterns.

**Parâmetros:**
- `pattern` (string, obrigatório): Glob pattern (ex: *.go, src/*/index.js)
- `path` (string, obrigatório): Caminho raiz para buscar
- `type` (string, opcional): Filtro por tipo (file, dir, all - padrão: all)
- `max_depth` (integer, opcional): Profundidade máxima (0 = sem limite - padrão: 0)
- `max_results` (integer, opcional): Máximo de resultados (padrão: 1000)

**Resposta:**
- `files`: Array de arquivos com path, type, size, modified
- `count`: Número total de arquivos encontrados
- `path`: Caminho raiz
- `total_size`: Tamanho total combinado

**Implementação:**
- Usa filepath.WalkDir com glob matching
- Suporta profundidade limitada para buscas eficientes

### 3. Symbols Tool (`symbols`)

Extrai símbolos de código (funções, classes, métodos, etc).

**Parâmetros:**
- `path` (string, obrigatório): Arquivo ou diretório
- `kinds` (array, opcional): Filtrar por tipo (function, class, method, variable, interface, type, const)
- `query` (string, opcional): Filtro por padrão de nome
- `max_results` (integer, opcional): Máximo de resultados (padrão: 500)

**Resposta:**
- `symbols`: Array de símbolos com name, kind, file, line, signature
- `count`: Número total de símbolos

**Linguagens Suportadas:**
- Go (.go)
- JavaScript/TypeScript (.js, .ts, .tsx, .jsx)
- Python (.py)
- Java (.java)

**Implementação:**
- Análise baseada em regex patterns
- Futura integração com Tree-sitter para maior precisão

### 4. References Tool (`references`)

Encontra referências a um identificador com word boundary matching.

**Parâmetros:**
- `symbol` (string, obrigatório): Nome do símbolo a buscar
- `path` (string, obrigatório): Caminho raiz para buscar
- `recursive` (boolean, opcional): Buscar recursivamente (padrão: true)
- `max_results` (integer, opcional): Máximo de resultados (padrão: 1000)

**Resposta:**
- `references`: Array de referências com file, line, column, context, kind
- `count`: Número total de referências
- `symbol`: Nome do símbolo

**Tipos de Referência:**
- `definition`: Definição do símbolo
- `import`: Importação/require
- `comment`: Em comentário
- `string`: Em string literal
- `usage`: Uso/referência

**Implementação:**
- Word boundary matching para precisão
- Análise de contexto para classificar tipo de referência

## Exemplo de Uso

```go
import "github.com/maylamcp/mayla/internal/tools/search"

tools := search.GetTools()

for _, tool := range tools {
    if tool.Name() == "search" {
        req := json.RawMessage(`{
            "pattern": "func",
            "path": "./",
            "recursive": true,
            "regex": true
        }`)

        result, err := tool.Execute(req)
        if err != nil {
            log.Fatal(err)
        }

        response := result.(*search.SearchResponse)
        for _, match := range response.Matches {
            fmt.Printf("%s:%d - %s\n", match.File, match.Line, match.Content)
        }
    }
}
```

## Performance

### Search Tool
- Com ripgrep: < 10ms para buscas em grandes codebases
- Fallback Go: < 100ms para buscas em projetos típicos
- Suporta até 1000 resultados por padrão

### Find Tool
- WalkDir otimizado: < 50ms para projetos típicos
- Glob matching eficiente
- Limite de profundidade para buscas focadas

### Symbols Tool
- Extração rápida via regex: < 100ms por arquivo
- Suporta múltiplas linguagens
- Tree-sitter será adicionado para precisão aumentada

### References Tool
- Word boundary matching: < 200ms para referências
- Classificação automática de tipo de referência
- Cache de símbolos planejado

## Limitações Conhecidas

1. **Symbols Tool**: Usa regex patterns, não é 100% preciso
   - Solução futura: integração com Tree-sitter

2. **References Tool**: Usa word boundaries simples
   - Pode gerar falsos positivos em certos contextos

3. **Ripgrep**: Opcional, fallback automático disponível
   - Instale com: `brew install ripgrep` (macOS) ou `cargo install ripgrep` (geral)

## Integração com MCP

As ferramentas implementam a interface `Tool` definida em `internal/tools/files/tools.go` e podem ser registradas no servidor MCP via:

```go
func GetAllTools() []Tool {
    fileTools := files.GetTools()
    searchTools := search.GetTools()
    return append(fileTools, searchTools...)
}
```
