# Referência Rápida - File Operations Tools

## Índice de Ferramentas

### read
Lê arquivo com detecção automática de encoding.
```json
{
  "path": "/absolute/path",
  "offset": 0,
  "limit": 0,
  "encoding": "auto"
}
```

### write
Escreve arquivo com escrita atômica.
```json
{
  "path": "/absolute/path",
  "content": "conteúdo",
  "createDirs": false,
  "backup": false
}
```

### edit
Edita arquivo com múltiplas operações.
```json
{
  "path": "/absolute/path",
  "edits": [
    {
      "startLine": 1,
      "endLine": 5,
      "newContent": "novo"
    }
  ]
}
```

### create
Cria arquivo ou diretório.
```json
{
  "path": "/absolute/path",
  "type": "file",
  "content": "inicial",
  "mode": "0644",
  "force": false
}
```

### delete
Deleta arquivo ou diretório.
```json
{
  "path": "/absolute/path",
  "recursive": false,
  "force": false
}
```

### move
Move ou renomeia.
```json
{
  "source": "/absolute/path1",
  "destination": "/absolute/path2",
  "overwrite": false
}
```

### list
Lista diretório.
```json
{
  "path": "/absolute/path",
  "recursive": false,
  "pattern": "*.txt",
  "showHidden": false,
  "sortBy": "name"
}
```

### info
Obtém informações detalhadas.
```json
{
  "path": "/absolute/path"
}
```

## Códigos de Erro Comuns

| Erro | Ferramenta(s) | Solução |
|------|---------------|--------|
| "path is required" | Todas | Informar `path` |
| "path does not exist" | read, edit, delete, move, info, list | Verificar caminho |
| "path already exists" | create, write | Use `force: true` |
| "failed to create file" | create, write | Verificar permissões |
| "invalid mode" | create | Mode deve ser octal válido |
| "invalid line range" | edit | Linhas devem estar dentro do arquivo |
| "directory not empty" | delete | Use `recursive: true` |
| "path is not a directory" | list | Deve ser diretório |
| "invalid request" | Todas | JSON inválido |

## Tipos de Resposta

### ReadResponse
```go
type ReadResponse struct {
    Content  string
    Size     int64
    Encoding string
    Lines    int
}
```

### WriteResponse
```go
type WriteResponse struct {
    Size    int64
    Path    string
    Backup  string
    Created bool
}
```

### EditResponse
```go
type EditResponse struct {
    Path      string
    Modified  bool
    Size      int64
    Lines     int
    EditsApplied int
}
```

### CreateResponse
```go
type CreateResponse struct {
    Path    string
    Type    string
    Created bool
    Size    int64
}
```

### DeleteResponse
```go
type DeleteResponse struct {
    Path    string
    Deleted bool
    Type    string
    Size    int64
}
```

### MoveResponse
```go
type MoveResponse struct {
    Source      string
    Destination string
    Type        string
    Size        int64
}
```

### ListResponse
```go
type ListResponse struct {
    Path  string
    Files []FileInfo
    Count int
}
```

### FileSystemInfo
```go
type FileSystemInfo struct {
    Path        string
    Name        string
    Type        string
    Size        int64
    Permissions string
    Mode        uint32
    Owner       string
    Created     time.Time
    Modified    time.Time
    Accessed    time.Time
    IsSymlink   bool
    FileCount   int
    TotalSize   int64
}
```

## Enum Values

### type (create)
- "file" - Criar arquivo
- "dir" - Criar diretório

### encoding (read)
- "utf-8" - UTF-8
- "utf-16" - UTF-16
- "iso-8859-1" - ISO-8859-1
- "auto" - Detectar automaticamente

### sortBy (list)
- "name" - Ordenar por nome
- "size" - Ordenar por tamanho
- "date" - Ordenar por data

### type (info response)
- "file" - Arquivo regular
- "dir" - Diretório
- "symlink" - Link simbólico

## Exemplos Rápidos

### Ler arquivo
```json
{"path": "/tmp/file.txt"}
```

### Escrever com backup
```json
{
  "path": "/tmp/file.txt",
  "content": "novo conteúdo",
  "backup": true
}
```

### Editar linha específica
```json
{
  "path": "/tmp/file.txt",
  "edits": [{"startLine": 10, "endLine": 10, "newContent": "linha"}]
}
```

### Buscar e substituir
```json
{
  "path": "/tmp/file.txt",
  "edits": [{"search": "old", "replace": "new"}]
}
```

### Listar Go files recursivamente
```json
{
  "path": "/project",
  "recursive": true,
  "pattern": "*.go"
}
```

### Deletar diretório com conteúdo
```json
{
  "path": "/tmp/dir",
  "recursive": true
}
```

## Limites

| Item | Limite |
|------|--------|
| Leitura máxima | 50MB |
| Path máximo | 4096 caracteres |
| Edições por chamada | Ilimitado |
| Listagem máxima | Ilimitado |

## Performance

| Operação | Tempo |
|----------|-------|
| read 1MB | < 5ms |
| write 1MB | < 10ms |
| edit | < 20ms |
| list 1000 arquivos | < 50ms |
| info | < 1ms |
| delete | < 5ms |
| move | < 5ms |

## Integração Go

```go
import (
    "encoding/json"
    "github.com/maylamcp/mayla/internal/registry"
)

func main() {
    reg := registry.NewRegistry()
    registry.InitializeAllTools(reg)

    input := map[string]interface{}{"path": "/tmp/file.txt"}
    data, _ := json.Marshal(input)
    result, err := reg.Execute("read", data)
    if err != nil {
        log.Fatal(err)
    }

    resp := result.(files.ReadResponse)
    fmt.Println(resp.Content)
}
```

## Padrões de Uso

### Backup antes de modificar
```json
{"path": "/file.txt", "backup": true}
```

### Operação segura (fail-safe)
```json
{"path": "/file.txt", "force": false}
```

### Criação recursiva
```json
{"path": "/deep/nested/dir", "type": "dir"}
```

### Listagem filtrada
```json
{
  "path": "/project",
  "pattern": "*.go",
  "sortBy": "date"
}
```

---

Para documentação completa, consulte `README.md`
