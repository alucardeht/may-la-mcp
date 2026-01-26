# YUME.md Implementation Plan

## Status: EM IMPLEMENTAÇÃO
## Updated: 2026-01-13

---

## CONCEITO

**YUME.md** = Arquivo OPCIONAL na raiz do projeto
- Contexto persistente para todas conversas daquele projeto
- Similar ao CLAUDE.md do Claude Code
- Injetado UMA VEZ por sessão (não em cada mensagem)

---

## ARQUITETURA

### 1. Injeção ÚNICA (Início da Sessão)

```
Workspace Abre
  ↓
Detecta YUME.md
  ↓
Carrega e Valida
  ↓
INJETA YUME.md UMA VEZ no Prompt
  ↓
Modelo mantém contexto na conversa
  ↓
Próximas mensagens: NÃO reenvia YUME.md
```

**Ordem de Injeção:**
1. BASE PROMPT (identidade)
2. YUME.md (contexto do projeto) ← INJETADO UMA VEZ
3. Skills trigadas
4. Mensagem do usuário

### 2. Hot Reload (Mudança Automática)

```
YUME.md é Editado/Salvo
  ↓
File Watcher Detecta (debounce 2s)
  ↓
Recarrega e Revalida
  ↓
INJETA YUME.md AUTOMATICAMENTE (sem botão manual)
  ↓
Mostra Toast: "YUME.md was updated" (2-3s, fade out)
  ↓
Conversa continua com novo contexto
```

**IMPORTANTE:**
- SEM botão de recarregar manual
- Detecta mudança e injeta AUTOMATICAMENTE
- Debounce 2s (anti-spam)
- Comparação de conteúdo (previous vs current) → só avisa se mudou

### 3. Re-injeção após Compactação

```
Conversa passa de X mensagens (ex: 50)
  ↓
Context window fica cheio (ex: 150K/200K tokens)
  ↓
COMPACTA/RESUMO da conversa
  ↓
INJETA YUME.md de novo
  ↓
Conversa continua mantendo YUME.md em contexto
```

### 4. RAG para Partes Específicas

```
Usuário pergunta sobre YUME.md
  ↓
Detecta: é sobre YUME.md?
  ↓
SIM → RAG extrai parte relevante
  ↓
Injeta SÓ a parte relevante (200 tokens, não 8K)
  ↓
ECONOMIA: 7.8K tokens salvos
```

---

## O QUE ESTÁ PRONTO

- Infraestrutura de skills funcionando
- Concatenação de system prompt
- Fluxo frontend → backend
- Tauri commands para filesystem
- WorkspaceContext (rootPath, files, tabs)
- TrustBadge (trust level do projeto)
- AnomalyBanner (problemas de segurança)

---

## O QUE FALTA IMPLEMENTAR

### Segurança (P0 - CRÍTICO)

**Rust Backend:**
- Path traversal validation
- Sandbox para run_command em projetos não-trusted
- Sistema de "trusted projects" (~/.yume/trusted-projects.json)
- Validação de tamanho (50KB max)
- Validação de encoding (UTF-8 apenas)
- Scanner de injection patterns

**Frontend:**
- Validação de symlinks

### Funcionalidade

**Novos Arquivos:**
- `services/yume-loader.ts` - Loader de YUME.md
- `components/YumeMdToast.tsx` - Toast de notificação
- `src-tauri/src/yume_md.rs` - Rust backend module

**Modificações:**
- `src/contexts/WorkspaceContext.tsx` - Adicionar yumeMd state
- `src/components/chat/ChatMessages.tsx` - Adicionar YumeMdToast
- `src/contexts/ChatContext.tsx` - Integração com injeção
- `src/types/workspace.types.ts` - Tipos YumeMdState, ValidationResult

**Funcionalidades:**
- Loader do YUME.md (detectar raiz do projeto, carregar, validar)
- File watcher para hot reload (debounce 2s, comparação de conteúdo)
- Injeção UMA VEZ no início da sessão
- Hot reload automático (injetar novamente quando YUME.md muda)
- Re-injeção após compactação da conversa
- RAG para partes específicas quando necessário

### UX

**Toast Component:**
- Posicionamento: topo do ChatMessages
- Duração: 2-3s, fade out (success), persistente (error/warning)
- Cores: verde (ok), amarelo (warning), vermelho (error)
- Sem botão "Reload session" (injeção automática)
- Só mostra se:
  - YUME.md mudou após estar ok
  - Estado mudou (ex: success → error)

**NÃO:**
- Chip constante no ChatInput
- Dropdown com preview
- Modal de confirmação para projetos novos

### Limites

- Tamanho máximo: 50KB
- Tokens máximos: 8,000
- Encoding: UTF-8 apenas

---

## VALIDAÇÕES DE SEGURANÇA

### Path Traversal
```rust
fn is_path_traversal(path: &str) -> bool {
    path.contains("..") || path.contains("//") || path.contains("\\")
}
```

### Sandbox (se untrusted)
```rust
fn enforce_sandbox(operation: Operation, trust_level: TrustLevel) -> bool {
    match trust_level {
        TrustLevel::Untrusted => operation.is_safe(),
        TrustLevel::Partial => operation.is_restricted_safe(),
        TrustLevel::Trusted => true,
    }
}
```

### Content Validation
```rust
fn validate_content(content: &str) -> Result<(), Error> {
    if content.len() > 50 * 1024 { // 50KB
        return Err(Error::FileTooLarge);
    }
    if !content.is_utf8() {
        return Err(Error::InvalidEncoding);
    }
    if content.contains("ignore previous instructions") {
        return Err(Error::InjectionDetected);
    }
    Ok(())
}
```

---

## NOTIFICAÇÃO INTELIGENTE (Anti-Spam)

### Quando Mostrar Toast

**SIM (mostrar toast) se:**
- `null → loaded` (primeira carga do workspace)
- `loaded → loaded` (conteúdo mudou REALMENTE)
- `loaded → error` (erro após estar ok)
- `loaded → warning` (warning após estar ok)
- `error → loaded` (corrigiu o erro)
- `warning → loaded` (corrigiu o warning)

**NÃO (não mostrar toast) se:**
- `loaded → loaded` (conteúdo idêntico)
- Salvar sem mudar conteúdo
- Debounce ainda ativo (2s)
- Usuario descartou (dismissed)

### Debounce (Anti-Spam)

```
Usuário digita "stack" → debounce timer reset
Usuário digita "stack: R" → debounce timer reset
Usuário digita "stack: Rea" → debounce timer reset
Usuário salva (Cmd+S) → debounce timer reset
Usuário para de digitar → espera 2s → dispara evento
```

Resultado: NÃO spam de notificações a cada edição

---

## ECONOMIA DE TOKENS

### Errado (O que NÃO fazer)
```
YUME.md em CADA mensagem
→ 8K tokens × 1000 msgs = 8M tokens desperdiçados
→ API cost absurdo
→ Performance terrível
```

### Correto (Como deve ser)
```
Injeção UMA VEZ no início
→ 8K tokens × 1 = 8K
→ Hot reload automático (quando YUME.md muda)
→ Re-injeção após compactação
→ RAG para partes específicas
→ ECONOMIA MASSIVA de tokens
```

---

## ESTRUTURA DE ARQUIVOS

### NOVOS
```
src/
  components/
    YumeMdToast.tsx           ← Toast component
  services/
    yume-loader.ts            ← Loader logic
  contexts/
    WorkspaceContext.tsx      ← Extender (adicionar yumeMd)
  types/
    workspace.types.ts        ← Adicionar YumeMdState, ValidationResult
src-tauri/src/
  yume_md.rs                  ← Rust backend
```

### MODIFICAR
```
src/
  components/
    chat/
      ChatMessages.tsx       ← Adicionar YumeMdToast
  contexts/
    ChatContext.tsx          ← Integração com injeção
```

---

## CRITÉRIOS DE SUCESSO

### Funcional
- [ ] YUME.md injetado UMA VEZ no início da sessão
- [ ] Modelo mantém contexto na conversa (não reenvia)
- [ ] File watcher detecta mudanças (debounce 2s)
- [ ] Hot reload: detecta mudança, INJETA AUTOMATICAMENTE (sem botão)
- [ ] Re-injeção após compactação da conversa
- [ ] RAG: extrai partes relevantes quando necessário
- [ ] Sessão longa: compacta mantendo YUME.md em contexto

### Segurança
- [ ] Path traversal bloqueado
- [ ] Sandbox enforcement se untrusted
- [ ] Symlinks não seguidos se inseguros
- [ ] Injection patterns detectados
- [ ] Tamanho máximo respeitado
- [ ] Encoding validado (UTF-8)

### Tokens/Economia
- [ ] NÃO reenvia YUME.md em cada mensagem
- [ ] YUME.md injetado UMA VEZ por sessão
- [ ] Hot reload automático (injeta quando YUME.md muda)
- [ ] RAG: só partes relevantes (não arquivo inteiro)
- [ ] Economia massiva de tokens

### UX/UI
- [ ] Toast só mostra se YUME.md mudou
- [ ] Toast SEM botão "Reload session" (injeção automática)
- [ ] NÃO spam de notificações (debounce + comparação)
- [ ] NÃO chip constante no ChatInput
- [ ] Posicionado no topo do ChatMessages
- [ ] Cores claras (verde/amarelo/vermelho)
- [ ] Smooth transitions (fade in/out)

---

## AGENTES CONSULTADOS

- architecture-enforcer (a05958c)
- ux-analyst (a3d66e2)
- security-guardian (a1a49d7)
- test-strategist (a459882)

---

## DATA ATUALIZAÇÃO

2026-01-13 - Hot reload automático (sem botão manual), re-injeção após compactação
