# Proposal: Preview em Tempo Real da Assinatura

**Change ID:** `add-realtime-signature-preview`
**Created:** 2026-08-14
**Status:** Implementation Complete
**Completed:** 2026-08-14

---

## Problem Statement

### Qual problema estamos resolvendo?

Hoje o aplicativo é uma caixa-preta: o usuário preenche seis campos de texto e
só descobre como a assinatura ficou **depois** de clicar em "Gerar Assinatura de
Email", sair da aplicação, localizar o PNG gerado no diretório de trabalho e
abri-lo em um visualizador externo.

### Quem é afetado?

Advogados e colaboradores do escritório que geram a própria assinatura — em
geral usuários não técnicos, executando o binário uma única vez.

### Qual é a dor atual?

1. **Ciclo de feedback longo.** Cada correção (um nome longo demais, um registro
   OAB errado, um telefone com formatação estranha) exige gerar → abrir → voltar
   ao app → corrigir → gerar de novo. O app nem sequer informa *onde* o arquivo
   foi salvo, apenas o nome.
2. **Estouro de texto invisível.** O modelo tem largura fixa (1604px) e as
   posições dos textos são coordenadas fixas. Um nome muito longo ou um campo
   "Profissão - Registro" extenso pode invadir a área do logo/telefones sem que
   nada avise o usuário.
3. **Arquivos descartados acumulados.** Cada tentativa deixa um
   `assinatura-<nome>.png` no diretório — o próprio repositório já tem um desses
   commitado por acidente.
4. **Janela subutilizada.** A janela de 400x300 mostra apenas o formulário; não
   há qualquer indicação visual do produto final.

## Proposed Solution

Transformar a janela em um editor WYSIWYG: renderizar a assinatura em memória a
cada alteração de qualquer campo e exibi-la em um `canvas.Image` dentro da
própria janela. O botão "Gerar" passa a ser apenas a etapa de **persistência** de
algo que o usuário já está vendo.

### Componentes-chave

1. **Separar renderização de persistência.** Extrair de
   `generateSignatureImage` uma função pura `renderSignature(dados) (image.Image, error)`
   que compõe e devolve a imagem. `generateSignatureImage` passa a ser
   `renderSignature` + `gg.SavePNG`. Uma única fonte de verdade garante que o
   preview e o arquivo salvo sejam **byte a byte idênticos**.

2. **Carregar assets uma única vez.** Hoje cada geração faz `gg.LoadPNG` do
   modelo e **quatro** chamadas a `LoadFontFace`, que reparseia arquivos TTF de
   275 KB e 750 KB do disco. Isso é aceitável uma vez por clique, mas inviável a
   cada tecla digitada. Os assets passam a ser carregados no startup para uma
   struct `renderer` (imagem de fundo + `font.Face` por tamanho), e
   `renderSignature` apenas desenha.

3. **Widget de preview.** Um `canvas.Image` com `FillMode = ImageFillContain` e
   `SetMinSize` respeitando a proporção do modelo (≈5,6:1). Atualizado via
   `img.Image = novaImagem; img.Refresh()`.

4. **Disparo reativo.** Cada `widget.Entry` recebe um `OnChanged` que chama um
   `atualizarPreview()` compartilhado. A renderização é síncrona na thread de UI
   — o Fyne 2.5.1 não expõe `fyne.Do`, então atualizar o canvas a partir de uma
   goroutine seria inseguro. Se a medição da Fase 4 mostrar latência perceptível
   ao digitar, aplicamos as mitigações descritas em *Risks*.

5. **Layout redesenhado.** Formulário e preview empilhados verticalmente
   (`container.NewBorder`), janela redimensionada para acomodar ambos.

### Resultado esperado

O usuário digita e vê a assinatura se formar em tempo real. Clica em "Gerar"
apenas quando estiver satisfeito, e é informado do caminho absoluto do arquivo.

## Scope

### In Scope

- Refatoração de `generateSignatureImage` em `renderSignature` (retorna
  `image.Image`) + persistência.
- Cache de assets (imagem de fundo e faces de fonte) carregados no startup.
- Widget `canvas.Image` de preview na janela principal.
- Handlers `OnChanged` em todos os seis campos.
- Novo layout e novo dimensionamento da janela.
- Preview inicial renderizado no startup (com os campos vazios / valor padrão do
  telefone), sem exigir interação.
- Mensagem de sucesso passa a exibir o caminho absoluto do arquivo salvo.
- Tratamento de erro do preview que não interrompe a digitação (erro exibido
  como rótulo, não como diálogo modal bloqueante).

### Out of Scope

- **Embutir assets com `go:embed`.** O app continua dependendo do diretório de
  trabalho para achar `modelo-assinatura.png` e as fontes. É uma fragilidade
  real e um bom próximo passo, mas ortogonal ao preview.
- **Detecção/ajuste automático de estouro de texto** (auto-shrink, quebra de
  linha, truncamento). O preview torna o problema *visível*; resolvê-lo é uma
  mudança separada.
- **Escolha do diretório de destino** via seletor de arquivos.
- **Zoom / pan no preview.**
- Alteração das coordenadas, cores, fontes ou do próprio modelo.
- Suporte a múltiplos modelos de assinatura.
- Testes automatizados de UI (o projeto não tem suíte de testes hoje; a Fase 1
  introduz apenas testes de unidade da camada de renderização).

## Impact Analysis

| Component | Change Required | Details |
|-----------|-----------------|---------|
| Database | Não | O projeto não tem persistência além do PNG de saída. |
| API | Não | Aplicação desktop offline, sem chamadas de rede. |
| State | Sim | Introduz um `renderer` com assets cacheados, vivo durante toda a sessão, e o acoplamento reativo campos → preview. |
| UI | Sim | Novo widget de preview, novo layout, nova geometria da janela, handlers `OnChanged`. |
| Renderização | Sim | `generateSignatureImage` dividida; carregamento de fontes migra de `LoadFontFace` (por chamada) para `font.Face` cacheada. |
| Build/CI | Não | Sem novas dependências diretas; `golang/freetype` e `golang.org/x/image` já são dependências indiretas via `gg`. |

## Architecture Considerations

### Encaixe nos padrões existentes

O projeto é um único `package main` com funções auxiliares no mesmo arquivo
(`juntarTelefones` é o precedente). A mudança mantém esse formato: nada de novos
pacotes enquanto o `main.go` couber confortavelmente em uma tela de leitura.

### Novos padrões introduzidos

1. **Struct `renderer` com assets cacheados.** Primeira vez que o projeto guarda
   estado entre operações. Justificativa: correção de desempenho, não
   arquitetura especulativa — reparsear 1 MB de TTF por tecla não é viável.

2. **Separação render/persist.** Padrão comum ("pure core, imperative shell") e
   pré-requisito para o preview: sem ele, preview e arquivo poderiam divergir.

3. **UI reativa via `OnChanged`.** O Fyne não tem data binding embutido em uso
   aqui; os seis handlers apontam para a mesma função.

### Dependências

Para cachear `font.Face` em vez de chamar `dc.LoadFontFace` a cada render,
usa-se `github.com/golang/freetype/truetype` + `golang.org/x/image/font`
diretamente com `dc.SetFontFace`. Ambos **já estão** no `go.sum` como
dependências indiretas do `gg`; promovê-las a diretas no `go.mod` não adiciona
nada novo à árvore de build.

### Nota sobre a alteração não commitada

O `main.go` atual já contém `juntarTelefones`, ainda não commitada. Esta proposta
assume esse código como ponto de partida.

## Success Criteria

- [ ] Digitar em qualquer um dos seis campos atualiza a imagem de preview sem
      ação adicional do usuário.
- [ ] A imagem exibida no preview é idêntica à salva em disco (mesma função de
      renderização; verificado comparando os bytes do PNG).
- [ ] A latência de atualização do preview é imperceptível ao digitar em ritmo
      normal (alvo: < 100 ms por render, medido por benchmark na Fase 4).
- [ ] O app abre já mostrando o preview do estado inicial, sem clique prévio.
- [ ] Um erro de renderização (ex.: `modelo-assinatura.png` ausente) é reportado
      de forma legível sem travar nem fechar o aplicativo.
- [ ] O diálogo de sucesso mostra o caminho absoluto do arquivo gerado.
- [ ] `go vet ./...` limpo e `go build` sem warnings.

## Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Render por tecla causa lag na digitação | Média | Alto | Cachear assets no startup (elimina o custo dominante: parse de TTF + decode de PNG). Se ainda houver lag, aplicar debounce com `time.AfterFunc` (~150 ms) e/ou renderizar em resolução reduzida no preview, mantendo o tamanho cheio só ao salvar. |
| Atualizar o canvas fora da thread de UI corrompe o render (Fyne 2.5.1 não tem `fyne.Do`) | Média | Alto | Manter a renderização síncrona no `OnChanged`. Se o debounce for necessário, o callback do timer deve marshalar a atualização para a thread de UI — não chamar `Refresh()` direto da goroutine. |
| Preview de 1604x287 escalado fica ilegível na janela | Alta | Médio | `ImageFillContain` + janela mais larga; o preview mostra o *enquadramento*, não serve para revisão em 100%. Documentar essa limitação. |
| Consumo de memória: uma `image.RGBA` de 1604x287 por render (~1,8 MB) | Baixa | Baixo | Sobrescrever a referência anterior a cada render; o GC recolhe. Evitar acumular histórico de previews. |
| Refatoração quebra a geração já funcional | Baixa | Alto | Fase 1 fecha com um teste que compara o PNG produzido pelo caminho refatorado com um golden file gerado pelo código atual. |
| Assets ausentes derrubam o app no startup (agora carregados cedo) | Média | Médio | Falha no carregamento não deve dar `panic`: exibir mensagem de erro na área do preview e desabilitar o botão "Gerar", mantendo a janela utilizável. |

---

## Archive Information

**Archived:** 2026-08-14
**Duration:** mesmo dia (proposta, implementação e archive em 2026-08-14)
**Outcome:** Successfully implemented

### Files Modified
- `main.go` — `renderer` com assets em cache, `render`/`save` separados,
  `montarPreview`, `juntarCampos`/`juntarProfissao`, layout `Border`
- `render_test.go` — **novo**: equivalência com a implementação anterior,
  oráculo `renderRef`, formatação das linhas compostas, benchmarks
- `ui_test.go` — **novo**: preview headless via `fyne.io/fyne/v2/test`
- `go.mod` — `golang.org/x/image` promovida a dependência direta
- `README.md` — uso, assets, modo degradado, comandos de desenvolvimento
- `.gitignore` — **novo**: PNGs gerados, binários, artefatos de teste
- `assinatura-email` — binário de 30 MB retirado do versionamento

### Specs Updated
- `openspec/specs/signature-rendering.md` — **criada**
- `openspec/specs/ui.md` — **criada**
- `openspec/project.md` — estrutura, convenções e coordenadas do modelo

### Resultados medidos
- Renderização: 9,00 ms → **2,27 ms** por imagem (3,4x); memória 11,5 MB → 2,1 MB
- Suíte: 14 testes de topo, `go vet` limpo, `gofmt` sem pendências

### Desvios em relação ao plano
- **Sem debounce.** A medição mostrou folga de 44x sobre o alvo de 100 ms, então
  a atualização ficou síncrona — o que também eliminou o risco de tocar no canvas
  fora da thread de UI (o Fyne 2.5.1 não tem `fyne.Do`).
- **Golden file substituído** por `legacyRender`/`renderRef`, referências vivas
  que não têm como ficar desatualizadas.
- **Altura do preview derivada** da proporção real do modelo, em vez do valor
  fixo de 125 px previsto.
- **Correção de fato:** as constantes `width = 2465` / `height = 439` do código
  original eram mortas e divergiam do modelo real (1604x287).
- **Escopo adicional a pedido do usuário:** omissão do separador ` - ` quando o
  registro não é informado (Fase 5).

### Próximos candidatos
- Embutir os assets com `go:embed`, hoje dependentes do diretório de trabalho
- Tratar o estouro de texto que o preview passou a evidenciar (tarefa 4.3)
- Rodar `go test` no CI, que hoje só compila
