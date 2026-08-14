# Implementation Tasks: Preview em Tempo Real da Assinatura

**Change ID:** `add-realtime-signature-preview`

---

## Phase 1: Foundation (Camada de Renderização)

Objetivo: `render` existe, é pura, é rápida, e produz exatamente a mesma imagem
que o código atual.

- [x] 1.1 ~~Gerar um **golden file** com o binário atual~~ **Substituído:** em
      vez de um PNG commitado, o `render_test.go` guarda `legacyRender`, uma
      cópia literal da implementação anterior. O teste compara as duas
      implementações em tempo de execução — mais forte que um golden file
      (não pode ficar desatualizado) e sem binário no repositório.
- [x] 1.2 Struct `signatureData` (name, email, profession, registration,
      telefones).
- [x] 1.3 Struct `renderer` com os assets cacheados: `bg *image.RGBA` e as
      faces `faceNome` / `faceCorpo` / `faceTelefones`.
- [x] 1.4 `newRenderer() (*renderer, error)`: carrega `modelo-assinatura.png`
      via `gg.LoadPNG` e as fontes via `gg.LoadFontFace` (função exportada do
      pacote — exatamente o que `dc.LoadFontFace` chamava por baixo, logo sem
      risco de divergência). O modelo é convertido para `*image.RGBA` uma vez.
- [x] 1.5 `(*renderer).render(d signatureData) image.Image`: copia o `bg`
      cacheado, `gg.NewContextForRGBA`, `SetHexColor`, `SetFontFace` por bloco.
      Não retorna erro — todo I/O já aconteceu em `newRenderer`.
- [x] 1.6 `generateSignatureImage` reescrita como
      `(*renderer).save(d, fileName) error` = `render` + `gg.SavePNG`.
- [x] 1.7 `golang.org/x/image` promovida a dependência direta no `go.mod`.
      `go mod tidy` executado: `go.sum` **inalterado** e o diff do `go.mod`
      ficou em uma única linha, confirmando que nada novo entrou na árvore de
      build. `github.com/golang/freetype` continua indireta, pois o parse ficou
      a cargo da `gg`.
- [x] 1.8 Testes de equivalência: `TestRenderEquivaleAImplementacaoAnterior`
      (pixel a pixel contra `legacyRender`), `TestRenderNaoReaproveitaBuffer`,
      `TestRenderEstavel` e `TestSaveGravaOMesmoQueRender`.
- [x] 1.9 `BenchmarkRender` (assets em cache) e `BenchmarkLegacyRender`
      (recarregando do disco), para quantificar o ganho.
- [x] 1.10 Removidas as constantes mortas `width = 2465` / `height = 439` —
      nunca usadas e divergentes do modelo real, que é 1604x287.
- [x] 1.11 Testes de `juntarTelefones` (que não tinha cobertura nenhuma).

**Quality Gate:** ✅ (Go 1.26.6)
- [x] `go vet ./...` limpo
- [x] `go test ./...` passa — 5 testes, incluindo a equivalência pixel a pixel
      com `legacyRender`
- [x] `go mod tidy` executado, `go.sum` inalterado
- [x] `BenchmarkRender` confirma < 100 ms por render — folga enorme:

      BenchmarkRender-12          2.70 ms/op    2.09 MB/op   57684 allocs/op
      BenchmarkLegacyRender-12    9.19 ms/op   11.54 MB/op   57882 allocs/op

      **3,4x mais rápido e 5,5x menos memória por render.** A contagem de
      alocações é praticamente idêntica: ela é dominada pela rasterização dos
      glifos, não pelo carregamento dos assets — o que o cache elimina é o
      volume de bytes, não o número de alocações.

---

## Phase 2: Business Logic (Estado e Coordenação)

Objetivo: existe um ponto único que lê os campos, renderiza e entrega a imagem —
sem ainda tocar em layout.

- [x] 2.1 `renderer` instanciado no início de `main()`, antes de montar a UI.
- [x] 2.2 Falha de `newRenderer` não causa `panic`: o erro é guardado em
      `rendErr`. **Nota:** por ora ele é reportado no clique do botão, via
      `dialog.ShowError` — exatamente o comportamento anterior. A UI em modo
      degradado (3.6) só faz sentido junto com o preview.
- [x] 2.3 O handler monta o `signatureData` diretamente a partir dos `Entry`,
      reaproveitando `juntarTelefones`. Uma função `coletarDados()` separada
      só se justifica quando houver um segundo chamador (o preview).
- [x] 2.3b `coletarDados()` extraída como closure em `main()` assim que ganhou
      o segundo chamador (o preview), conforme previsto em 2.3.
- [x] 2.4 `atualizar()` dentro de `montarPreview`: `coletarDados` →
      `renderer.render` → `imagem.Image = ...` → `Refresh()`.
- [x] 2.5 Handler do botão "Gerar" usando `renderer.save`, com a validação de
      campos obrigatórios (nome, email, profissão) preservada e mantida **antes**
      da checagem de `rendErr`, para que a ordem das mensagens de erro não mude.
- [x] 2.6 Caminho absoluto via `filepath.Abs` no diálogo de sucesso. Se a
      resolução falhar, cai para o nome relativo — o arquivo já foi salvo, não
      faz sentido reportar erro nesse ponto.

**Quality Gate:** ✅
- [x] `go vet ./...` limpo
- [x] `go build` bem-sucedido
- [x] Fluxo de geração verificado: `TestSaveGravaOMesmoQueRender` e
      `TestPreviewCorrespondeAoArquivoSalvo` cobrem a imagem produzida; smoke
      test da GUI sob WSLg confirma que o app abre e se mantém rodando.

---

## Phase 3: User Interface

Tudo isso ficou em `montarPreview`, que devolve a área de preview já ligada aos
campos — ou a mensagem de erro, no modo degradado. Extrair a função manteve o
`main()` legível e, de quebra, tornou o preview testável sem abrir janela.

- [x] 3.1 Widget de preview: `canvas.NewImageFromImage(...)` com
      `FillMode = canvas.ImageFillContain`. A altura do `SetMinSize` é
      **derivada dos bounds reais do modelo** (`previewLargura * Dy/Dx`) em vez
      do 125 fixo do plano — assim não desalinha se o modelo mudar.
- [x] 3.2 `OnChanged` registrado nos seis campos, todos chamando o mesmo
      `atualizar`.
- [x] 3.3 Layout `container.NewBorder`: formulário no topo, preview no centro.
- [x] 3.4 Janela em 760x480 (constantes `janelaLargura` / `janelaAltura`).
- [x] 3.5 Preview inicial renderizado na própria construção do widget — o app
      abre já mostrando o modelo. Coberto por `TestPreviewInicialJaRenderizado`.
- [x] 3.6 Modo degradado: `widget.NewLabel` com o erro e a lista de arquivos
      esperados, no lugar do preview, e `generateButton.Disable()`. Coberto por
      `TestModoDegradado`.
- [x] 3.7 Renomeada a variável local `canvas` em `render()` para `dst` — ela
      sombreava o pacote `fyne.io/fyne/v2/canvas`, recém-importado.

**Quality Gate:** ✅
- [x] `go vet ./...` limpo e `gofmt` sem pendências
- [x] Atualização ao digitar coberta por `TestPreviewAtualizaAoDigitar` e
      `TestPreviewVoltaAoLimparCampo` (headless, via `fyne.io/fyne/v2/test`) —
      mais forte que a inspeção manual prevista no plano, porque não regride
- [x] Modo degradado verificado de duas formas: `TestModoDegradado` e execução
      real do binário a partir de um diretório sem os assets, sob WSLg — a
      janela abre e permanece aberta em vez de crashar

---

## Phase 4: Integration & Polish

- [x] 4.1 Latência medida: **2,27 ms por render** (`BenchmarkRender`, 50
      iterações), contra ~50–150 ms de intervalo entre teclas em digitação
      normal. **Debounce não foi implementado** — seria complexidade sem
      benefício, e evitá-lo elimina de vez o risco de tocar no canvas fora da
      thread de UI (o Fyne 2.5.1 não tem `fyne.Do`). A atualização é síncrona
      no `OnChanged`.
- [x] 4.2 Equivalência preview × arquivo coberta por
      `TestPreviewCorrespondeAoArquivoSalvo`: digita no campo, grava o PNG, relê
      do disco e compara pixel a pixel com a imagem exibida no preview.
- [x] 4.3 Estouro de texto verificado visualmente no archive: com
      "Maria Fernanda Albuquerque dos Santos Vasconcelos Filha" + profissão
      longa, o texto invade o bloco de telefones/endereço e fica **claramente
      visível** antes de gerar o arquivo — que era o objetivo da tarefa.
      Corrigir o estouro (auto-shrink/quebra) segue fora de escopo e vira
      candidato a uma próxima proposta.
- [x] 4.4 `.gitignore` criado (`assinatura-*.png`, binários, artefatos de
      teste) e `git rm --cached assinatura-email` para tirar do versionamento o
      binário de 30 MB. **Correção do plano:** o
      `assinatura-Fabiano Chiaretto Fernandes.png` nunca chegou a ser commitado
      — estava apenas untracked, e você já o removeu. O CI compila
      `assinatura-email.exe` do zero, então destrackear não afeta o build.
- [x] 4.5 `README.md` reescrito: uso, tabela de assets exigidos no diretório de
      trabalho, comportamento em modo degradado e comandos de desenvolvimento.
- [x] 4.6 `openspec/project.md` sincronizado: novos arquivos de teste, ordem do
      `main.go`, e as convenções que a mudança estabeleceu (assets carregados
      uma vez, render como fonte única de verdade, `renderer` não concorrente).

**Quality Gate:** ✅
- [x] Todos os testes passam (10 testes)
- [x] `go vet ./...` limpo e `gofmt -l` vazio
- [x] Success Criteria verificados (ver abaixo)
- [x] Documentação sincronizada (README + project.md + specs)

### Success Criteria da proposta

| Critério | Status | Evidência |
|----------|--------|-----------|
| Digitar em qualquer campo atualiza o preview | ✅ | `TestPreviewAtualizaAoDigitar`, `TestPreviewVoltaAoLimparCampo` |
| Preview idêntico ao arquivo salvo | ✅ | `TestPreviewCorrespondeAoArquivoSalvo` (pixel a pixel) |
| Latência < 100 ms por render | ✅ | 2,27 ms — 44x de folga |
| App abre já com o preview | ✅ | `TestPreviewInicialJaRenderizado` |
| Erro de render não trava o app | ✅ | `TestModoDegradado` + execução real sem assets |
| Diálogo mostra caminho absoluto | ✅ | `filepath.Abs` no handler |
| `go vet` limpo e build sem warnings | ✅ | quality gate acima |

---

## Phase 5: Ajuste de formatação (registro opcional)

Pedido do usuário após a implementação: sem registro informado, não desenhar o
` - ` na tela.

- [x] 5.1 Extrair `juntarCampos(esquerda, direita, separador)` a partir de
      `juntarTelefones` — a regra "omitir o separador quando um lado está
      vazio" já existia, só não estava reaproveitável.
- [x] 5.2 `juntarProfissao(profession, registration)` sobre esse helper, e
      `render` passa a usá-la no lugar de `d.profession+" - "+d.registration`.
- [x] 5.3 `renderRef(tb, d, linhaProfissao)` nos testes: oráculo que recebe o
      texto esperado literalmente, permitindo afirmar no pixel qual linha foi
      desenhada. `legacyRender` foi reescrita sobre ele.
- [x] 5.4 Testes: `TestJuntarProfissao` (6 casos), `TestRenderLinhaProfissao`
      (5 casos, verificação em pixel), `TestRenderSemRegistroDivergeDoComportamentoAntigo`
      e `TestPreviewRemoveTracoAoApagarRegistro` (cenário de UI).
- [x] 5.5 Specs e docs sincronizados: novo requisito em
      `signature-rendering_delta.md`, cenário em `ui_delta.md`, ressalva na
      equivalência com o comportamento antigo, README e `project.md`.

**Quality Gate:** ✅
- [x] `gofmt -l` vazio, `go vet ./...` limpo
- [x] `go test ./...` passa — 14 testes

---

## Completion Checklist

- [x] Todas as fases concluídas
- [x] Todos os quality gates aprovados
- [x] Documentação sincronizada
- [x] Pronto para `/openspec-archive`

Nenhum item em aberto. A 4.3, última pendência, foi verificada visualmente
durante o archive.
