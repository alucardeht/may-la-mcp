# File Operations Tools

Suíte completa de ferramentas de operações de arquivo para o May-la MCP server.

## Ferramentas Disponíveis

### 1. **read** - Leitura de Arquivo
Lê conteúdo de arquivo com detecção automática de encoding e suporte a streaming.

**Parâmetros:**
- `path` (string, obrigatório): Caminho absoluto do arquivo
- `offset` (integer): Offset em bytes (padrão: 0)
- `limit` (integer): Máximo de bytes a ler (0 = sem limite)
- `encoding` (string): utf-8, utf-16, iso-8859-1, auto

**Resposta:**
- `content`: Conteúdo do arquivo
- `size`: Tamanho total do arquivo em bytes
- `encoding`: Encoding detectado/utilizado
- `lines`: Número de linhas

**Exemplo:**
```json
{
  "path": "/absolute/path/to/file.txt",
  "limit": 1000
}
```

### 2. **write** - Escrita de Arquivo
Escreve conteúdo em arquivo com operações atômicas.

**Parâmetros:**
- `path` (string, obrigatório): Caminho absoluto do arquivo
- `content` (string, obrigatório): Conteúdo a escrever
- `createDirs` (boolean): Criar diretórios pai (padrão: false)
- `backup` (boolean): Criar backup .bak antes de sobrescrever (padrão: false)

**Resposta:**
- `size`: Tamanho do arquivo escrito
- `path`: Caminho do arquivo
- `backup`: Caminho do backup criado (se aplicável)
- `created`: Se é um novo arquivo

**Exemplo:**
```json
{
  "path": "/absolute/path/to/file.txt",
  "content": "novo conteúdo",
  "backup": true
}
```

### 3. **edit** - Edição de Arquivo
Edita arquivo com múltiplas operações (linha ou busca/substituição).

**Parâmetros:**
- `path` (string, obrigatório): Caminho absoluto do arquivo
- `edits` (array, obrigatório): Array de operações
  - `startLine`/`endLine`: Range de linhas (1-indexed)
  - `newContent`: Novo conteúdo
  - OU `search`/`replace`: Buscar e substituir texto

**Resposta:**
- `path`: Caminho do arquivo
- `modified`: Se foi modificado
- `size`: Novo tamanho
- `lines`: Novo número de linhas
- `editsApplied`: Quantas edições foram aplicadas

**Exemplo:**
```json
{
  "path": "/absolute/path/to/file.txt",
  "edits": [
    {
      "startLine": 10,
      "endLine": 15,
      "newContent": "nova linha"
    }
  ]
}
```

### 4. **create** - Criação de Arquivo/Diretório
Cria novo arquivo ou diretório.

**Parâmetros:**
- `path` (string, obrigatório): Caminho absoluto
- `type` (string, obrigatório): "file" ou "dir"
- `content` (string): Conteúdo inicial (somente para arquivo)
- `mode` (string): Permissões octal (ex: "0644")
- `force` (boolean): Sobrescrever se existir (padrão: false)

**Resposta:**
- `path`: Caminho criado
- `type`: Tipo criado
- `created`: Se foi criado com sucesso
- `size`: Tamanho do arquivo (0 para dir)

**Exemplo:**
```json
{
  "path": "/absolute/path/to/newfile.txt",
  "type": "file",
  "content": "conteúdo inicial",
  "mode": "0644"
}
```

### 5. **delete** - Deleção de Arquivo/Diretório
Deleta arquivo ou diretório.

**Parâmetros:**
- `path` (string, obrigatório): Caminho absoluto
- `recursive` (boolean): Deletar recursivamente (padrão: false)
- `force` (boolean): Forçar deleção (padrão: false)

**Resposta:**
- `path`: Caminho deletado
- `deleted`: Sucesso
- `type`: Tipo deletado
- `size`: Tamanho deletado

**Exemplo:**
```json
{
  "path": "/absolute/path/to/file.txt"
}
```

### 6. **move** - Mover/Renomear
Move ou renomeia arquivo/diretório.

**Parâmetros:**
- `source` (string, obrigatório): Caminho de origem
- `destination` (string, obrigatório): Caminho de destino
- `overwrite` (boolean): Sobrescrever se existir (padrão: false)

**Resposta:**
- `source`: Caminho original
- `destination`: Novo caminho
- `type`: Tipo movido
- `size`: Tamanho

**Exemplo:**
```json
{
  "source": "/absolute/path/old.txt",
  "destination": "/absolute/path/new.txt",
  "overwrite": false
}
```

### 7. **list** - Listar Diretório
Lista conteúdo de diretório com filtros e ordenação.

**Parâmetros:**
- `path` (string, obrigatório): Caminho do diretório
- `recursive` (boolean): Listar recursivamente (padrão: false)
- `pattern` (string): Filtro glob (ex: "*.go")
- `showHidden` (boolean): Mostrar arquivos ocultos (padrão: false)
- `sortBy` (string): "name", "size", "date" (padrão: "name")

**Resposta:**
- `path`: Diretório listado
- `files`: Array de arquivos
  - `name`: Nome do arquivo
  - `path`: Caminho absoluto
  - `type`: "file" ou "dir"
  - `size`: Tamanho em bytes
  - `modified`: Data de modificação
  - `permissions`: String de permissões
- `count`: Total de arquivos

**Exemplo:**
```json
{
  "path": "/absolute/path/to/dir",
  "pattern": "*.go",
  "recursive": true,
  "sortBy": "date"
}
```

### 8. **info** - Informações de Arquivo/Diretório
Retorna informações detalhadas sobre arquivo ou diretório.

**Parâmetros:**
- `path` (string, obrigatório): Caminho absoluto

**Resposta:**
- `path`: Caminho
- `name`: Nome do arquivo
- `type`: "file", "dir", "symlink"
- `size`: Tamanho em bytes
- `permissions`: String de permissões
- `mode`: Modo octal
- `owner`: Proprietário
- `created`: Data de criação
- `modified`: Data de modificação
- `accessed`: Data de acesso
- `isSymlink`: Se é symlink
- `fileCount`: Número de arquivos (somente dir)
- `totalSize`: Tamanho total (somente dir)

**Exemplo:**
```json
{
  "path": "/absolute/path/to/file.txt"
}
```

## Integração no Registry

Para integrar essas ferramentas no MCP server, adicione ao seu registry:

```go
package main

import (
  "github.com/maylamcp/mayla/internal/tools/files"
)

func registerFileTools(registry *ToolRegistry) {
  for _, tool := range files.GetTools() {
    registry.Register(tool)
  }
}
```

## Características

- ✅ Operações atômicas para write (temp file + rename)
- ✅ Detecção automática de encoding (UTF-8, UTF-16, ISO-8859-1)
- ✅ Suporte a múltiplas edições em uma chamada
- ✅ Listagem recursiva com filtros glob
- ✅ Backup automático antes de sobrescrever
- ✅ Tratamento robusto de erros
- ✅ Schema JSON completo para cada ferramenta
- ✅ Testes unitários inclusos

## Erros Comuns

1. **"path does not exist"** - Caminho inválido ou não encontrado
2. **"path is not a directory"** - Operação esperava diretório mas recebeu arquivo
3. **"directory not empty"** - Use `recursive: true` ao deletar diretório com conteúdo
4. **"path already exists"** - Use `force: true` para sobrescrever
5. **"invalid line range"** - Linhas fora do intervalo válido do arquivo
