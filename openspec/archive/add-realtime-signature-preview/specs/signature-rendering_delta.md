# Delta: Renderização da Assinatura

**Change ID:** `add-realtime-signature-preview`
**Affects:** `main.go` (`generateSignatureImage`), carregamento de assets, `go.mod`

---

## ADDED

### Requirement: Renderização em memória desacoplada da gravação em disco

Existe uma operação que compõe a assinatura e devolve uma `image.Image`, sem
tocar no sistema de arquivos. A gravação passa a ser uma etapa separada que
consome essa imagem.

#### Scenario: Render puro
- GIVEN um conjunto de dados de assinatura e um renderer inicializado
- WHEN a renderização é solicitada
- THEN uma `image.Image` composta é retornada
- AND nenhum arquivo é criado, lido ou modificado no disco

#### Scenario: Fonte única de verdade entre preview e arquivo
- GIVEN os mesmos dados de entrada
- WHEN a imagem é renderizada para o preview e depois salva em disco
- THEN os dois resultados são pixel a pixel idênticos, por usarem exatamente o
  mesmo caminho de renderização

---

### Requirement: Assets carregados uma única vez no startup

A imagem de fundo e as faces de fonte são carregadas e parseadas uma vez,
durante a inicialização, e reutilizadas em todas as renderizações da sessão.

#### Scenario: Custo de I/O amortizado
- GIVEN um renderer inicializado com sucesso
- WHEN N renderizações consecutivas são executadas
- THEN `modelo-assinatura.png` é lido exatamente 1 vez
- AND cada arquivo de fonte é parseado exatamente 1 vez
- AND nenhuma renderização subsequente acessa o disco

#### Scenario: Desempenho por renderização
- GIVEN um renderer inicializado
- WHEN uma renderização é medida por benchmark
- THEN o tempo por renderização é inferior a 100 ms

#### Scenario: Falha de carregamento é reportada, não fatal
- GIVEN que um dos assets está ausente ou corrompido
- WHEN a inicialização do renderer é tentada
- THEN um `error` descritivo, identificando o asset, é retornado
- AND nenhum `panic` ocorre

---

## MODIFIED

### Requirement: Composição da imagem de assinatura

Antes: `generateSignatureImage(name, email, profession, registration, celular, fileName string) error`
carregava o PNG de fundo, chamava `dc.LoadFontFace` quatro vezes (reparseando os
arquivos TTF a cada chamada), desenhava os textos e salvava o arquivo — tudo em
uma única função, com cinco parâmetros posicionais de string.

Agora: os dados de entrada são agrupados em uma struct; o carregamento de assets
vive na inicialização do renderer; a composição usa `dc.SetFontFace` com faces
já em memória; e a gravação é uma etapa separada.

O **resultado visual permanece inalterado**: mesmas coordenadas, mesmos tamanhos
de fonte, mesma cor `#4C4C4C`, mesmas dimensões de saída.

#### Scenario: Equivalência com o comportamento anterior
- GIVEN um conjunto fixo de dados de assinatura **com registro preenchido**
- AND uma referência produzida pela implementação anterior com esses dados
- WHEN a implementação refatorada compõe a imagem
- THEN o resultado é idêntico pixel a pixel
- AND a única divergência intencional em relação ao comportamento antigo é a
  omissão do separador quando o registro está vazio (ver requisito abaixo)

#### Scenario: Posicionamento dos textos preservado
- GIVEN dados de assinatura completos
- WHEN a imagem é composta
- THEN o nome é desenhado em Arial 40 na posição (300, 162)
- AND a linha de profissão em Arial 30 na posição (300, 198)
- AND o e-mail em Arial 30 na posição (300, 265)
- AND a linha de telefones em Arial Bold 25 na posição (900, 148)
- AND todos na cor `#4C4C4C`

---

### Requirement: Separador omitido quando um dos lados está vazio

A linha de profissão é montada por `juntarProfissao`, que só insere ` - ` se
ambos os lados tiverem conteúdo. O registro é opcional (não entra na validação
de campos obrigatórios), então a linha nunca pode exibir um traço órfão.

A regra é a mesma já aplicada aos telefones por `juntarTelefones`; as duas
compartilham o helper `juntarCampos`, que difere apenas no separador.

#### Scenario: Registro não informado
- GIVEN profissão "Advogado" e registro vazio
- WHEN a linha de profissão é composta
- THEN o texto desenhado é exatamente "Advogado"
- AND não há traço nem espaço sobrando ao final

#### Scenario: Registro só com espaços
- GIVEN profissão "Advogado" e registro "   "
- WHEN a linha de profissão é composta
- THEN o texto desenhado é exatamente "Advogado"

#### Scenario: Profissão não informada
- GIVEN profissão vazia e registro "OAB/MG - 123456"
- WHEN a linha de profissão é composta
- THEN o texto desenhado é exatamente "OAB/MG - 123456"
- AND a linha não começa com um traço

#### Scenario: Ambos vazios
- GIVEN profissão e registro vazios
- WHEN a linha de profissão é composta
- THEN nada é desenhado naquela linha

---

### Requirement: Declaração de dependências

`golang.org/x/image` passa de dependência indireta (via `gg`) a dependência
direta, por fornecer o tipo `font.Face` usado nos campos do renderer.

O parse das fontes continua sendo feito pela própria `gg.LoadFontFace`
(função exportada no nível do pacote), e **não** por `truetype.Parse` direto —
é literalmente o mesmo código que o `dc.LoadFontFace` anterior chamava, o que
elimina qualquer risco de divergência no resultado. Por isso
`github.com/golang/freetype` permanece uma dependência indireta.

#### Scenario: Árvore de build inalterada
- GIVEN o `go.mod` atualizado com a dependência promovida
- WHEN `go mod tidy` é executado
- THEN nenhum módulo novo é adicionado ao `go.sum`
- AND apenas o marcador `// indirect` é removido da entrada de
  `golang.org/x/image`

---

## REMOVED

### Requirement: Carregamento de fonte por chamada de desenho

As chamadas a `dc.LoadFontFace("./Arial.ttf", ...)` intercaladas com o desenho
dos textos deixam de existir. Reparsear ~1 MB de arquivos TrueType a cada
composição era tolerável quando a composição ocorria uma vez por clique, mas
inviabiliza a renderização por tecla digitada.
