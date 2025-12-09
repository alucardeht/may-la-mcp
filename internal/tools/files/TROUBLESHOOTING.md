# Troubleshooting - File Operations Tools

## Problemas Comuns

### 1. "path is required"
**Causa:** Campo `path` não foi informado

**Solução:**
```json
{
  "path": "/absolute/path/to/file.txt"
}
```

---

### 2. "path does not exist"
**Causa:** Arquivo ou diretório não encontrado

**Verificações:**
- [ ] Caminho está correto?
- [ ] Arquivo foi criado antes de ler/editar?
- [ ] Permissões de acesso?
- [ ] Caminho absoluto?

**Exemplo:**
```bash
# Verificar se arquivo existe
ls -la /absolute/path/to/file.txt
```

---

### 3. "path already exists"
**Causa:** Tentando criar arquivo que já existe

**Soluções:**
1. Use `force: true` para sobrescrever
2. Use ferramenta `move` para renomear
3. Delete primeiro

```json
{
  "path": "/absolute/path/file.txt",
  "type": "file",
  "force": true
}
```

---

### 4. "directory not empty"
**Causa:** Tentando deletar diretório com conteúdo

**Solução:** Use `recursive: true`

```json
{
  "path": "/absolute/path/dir",
  "recursive": true
}
```

---

### 5. "failed to create file"
**Causa:** Permissão negada ou diretório pai não existe

**Soluções:**
1. Verificar permissões do diretório pai
2. Criar diretório pai manualmente
3. Usar caminhos com permissões adequadas

```bash
# Verificar permissões
ls -ld /absolute/path/

# Criar diretório pai
mkdir -p /absolute/path/
chmod 755 /absolute/path/
```

---

### 6. "invalid line range"
**Causa:** startLine ou endLine fora do intervalo válido

**Verificar:**
```go
// Ler arquivo para saber quantas linhas tem
result, _ := readTool.Execute(readData)
resp := result.(ReadResponse)
fmt.Printf("Total de linhas: %d\n", resp.Lines)
```

**Corrigir:**
```json
{
  "path": "/file.txt",
  "edits": [
    {
      "startLine": 1,
      "endLine": 5,
      "newContent": "novo"
    }
  ]
}
```

---

### 7. "path is not a directory"
**Causa:** Ferramenta esperava diretório mas recebeu arquivo

**Solução:** Verificar se é diretório

```go
result, _ := infoTool.Execute(infoData)
resp := result.(FileSystemInfo)
if resp.Type != "dir" {
  // Não é diretório
}
```

---

### 8. "invalid request"
**Causa:** JSON inválido

**Verificações:**
- [ ] JSON bem formado?
- [ ] Aspas duplas em chaves?
- [ ] Tipos de dados corretos?

**Teste:**
```bash
# Validar JSON
echo '{"path": "/file.txt"}' | jq .
```

---

### 9. "invalid mode"
**Causa:** Modo (permissões) inválido

**Valores válidos:** Octal (ex: 0644, 0755)

```json
{
  "path": "/file.txt",
  "type": "file",
  "mode": "0644"
}
```

---

### 10. "failed to move"
**Causa:** Erro ao renomear ou mover para outro filesystem

**Soluções:**
1. Verificar se source existe
2. Verificar se destination é válida
3. Usar `overwrite: true` se destino existe

```bash
# Testar move no shell
mv /source/file /destination/file
```

---

## Erros de Encoding

### "Invalid UTF-8 sequence"
**Causa:** Arquivo com encoding diferente

**Solução:** Especificar encoding correto

```json
{
  "path": "/file.txt",
  "encoding": "iso-8859-1"
}
```

---

## Problemas de Performance

### "Operação lenta demais"
**Causa:** Arquivo muito grande ou muitos arquivos

**Soluções:**
1. Use `limit` em read para ler apenas parte
2. Use `pattern` em list para filtrar
3. Não use `recursive: true` se não precisar

```json
{
  "path": "/large/file.txt",
  "limit": 1000
}
```

---

## Problemas de Permissão

### "Permission denied"
**Causa:** Sem permissão para ler/escrever

**Soluções:**
```bash
# Verificar permissões
ls -la /path/to/file

# Mudar permissões
chmod 644 /path/to/file
chmod 755 /path/to/dir

# Verificar owner
chown $USER /path/to/file
```

---

## Debugging

### Ativar Logging
```go
import "log"

log.SetFlags(log.LstdFlags | log.Lshortfile)
log.Println("Debug info")
```

### Validar Input
```go
// Verificar JSON antes de executar
var req files.ReadRequest
json.Unmarshal(data, &req)
fmt.Printf("Path: %s\n", req.Path)
```

### Verificar Resposta
```go
result, err := reg.Execute("read", data)
if err != nil {
  log.Printf("Error: %v", err)
} else {
  resp := result.(files.ReadResponse)
  fmt.Printf("Content: %s\n", resp.Content)
}
```

---

## Casos Especiais

### Symlinks
Info detecta symlinks automaticamente:
```go
info := result.(FileSystemInfo)
if info.IsSymlink {
  // É um link simbólico
}
```

### Diretórios Vazios
Delete falha em diretório vazio a menos que use force:
```json
{
  "path": "/empty/dir",
  "force": true
}
```

### Caracteres Especiais
Caminhos com espaços funcionam normalmente:
```json
{
  "path": "/path/with spaces/file.txt"
}
```

### Arquivos Grandes
Limite de 50MB em read. Para arquivos maiores:
1. Use limit para ler em chunks
2. Use ferramentas stream específicas

```json
{
  "path": "/huge/file.bin",
  "limit": 1048576
}
```

---

## Testes para Validação

### Teste Básico
```bash
# Criar arquivo de teste
echo "test content" > /tmp/test.txt

# Testar read
curl -X POST http://localhost:8000/tools/read \
  -d '{"path": "/tmp/test.txt"}'
```

### Teste de Edit
```bash
# Verificar antes
cat /tmp/test.txt

# Editar
curl -X POST http://localhost:8000/tools/edit \
  -d '{
    "path": "/tmp/test.txt",
    "edits": [{"search": "test", "replace": "modified"}]
  }'

# Verificar depois
cat /tmp/test.txt
```

---

## Checklist de Troubleshooting

- [ ] Caminho está em formato absoluto?
- [ ] Arquivo/diretório existe?
- [ ] Tem permissões corretas?
- [ ] JSON está bem formado?
- [ ] Tipos de dados estão corretos?
- [ ] Encoding está correto?
- [ ] Recurso está disponível (espaço disco)?
- [ ] Registry foi inicializado?
- [ ] Tool foi registrada?

---

## Suporte Adicional

Para problemas não listados aqui:

1. **Documentação:** Consulte README.md
2. **Referência:** Consulte REFERENCE.md
3. **Exemplos:** Veja examples/file_operations.go
4. **Testes:** Execute `make test` para validação
5. **Código:** Examine as implementações em *.go

---

## FAQ Rápido

**P: Por que preciso usar caminhos absolutos?**
R: Para segurança e evitar ambiguidade.

**P: Posso editar arquivo enquanto está em uso?**
R: Sim, mas não é recomendado para arquivos abertos.

**P: Quantas edições posso fazer em uma chamada?**
R: Ilimitado, mas cuidado com performance.

**P: O backup é obrigatório?**
R: Não, é opcional com `backup: true`.

**P: Posso deletar arquivos do sistema?**
R: Sim, cuidado com o `path`!

---

**Última atualização:** 2025-12-09
