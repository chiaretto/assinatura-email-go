# Renderização da Assinatura

Fonte de verdade sobre como a imagem da assinatura é composta e gravada.

**Última atualização:** 2026-08-14 (change `add-realtime-signature-preview`)

---

## Requirements

### Requirement: Renderização em memória desacoplada da gravação em disco

Existe uma operação que compõe a assinatura e devolve uma `image.Image`, sem
tocar no sistema de arquivos. A gravação é uma etapa separada que consome essa
imagem.

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

#### Scenario: Cada renderização devolve um buffer próprio
- GIVEN uma imagem já devolvida por uma renderização anterior
- WHEN uma nova renderização é executada
- THEN a imagem anterior permanece intacta
- AND o modelo em cache não acumula os textos desenhados

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

  *Medido em 2026-08-14: 2,27 ms por render, contra 9,00 ms do caminho anterior
  que recarregava os assets — 3,4x mais rápido e 5,5x menos memória.*

#### Scenario: Falha de carregamento é reportada, não fatal
- GIVEN que um dos assets está ausente ou corrompido
- WHEN a inicialização do renderer é tentada
- THEN um `error` descritivo, identificando o asset, é retornado
- AND nenhum `panic` ocorre

#### Scenario: Renderer não é seguro para uso concorrente
- GIVEN que as `font.Face` da `gg` não podem ser usadas em paralelo entre
  goroutines
- WHEN a renderização for chamada fora da thread de UI
- THEN o acesso precisa ser serializado pelo chamador

---

### Requirement: Composição da imagem de assinatura

Os dados de entrada são agrupados em uma struct; o carregamento de assets vive
na inicialização do renderer; a composição usa `dc.SetFontFace` com faces já em
memória; e a gravação é uma etapa separada.

#### Scenario: Posicionamento dos textos
- GIVEN dados de assinatura completos
- WHEN a imagem é composta
- THEN o nome é desenhado em Arial 40 na posição (300, 162)
- AND a linha de profissão em Arial 30 na posição (300, 198)
- AND o e-mail em Arial 30 na posição (300, 265)
- AND a linha de telefones em Arial Bold 25 na posição (900, 148)
- AND todos na cor `#4C4C4C`
- AND sobre `modelo-assinatura.png`, de 1604x287 pixels

#### Scenario: Equivalência com o comportamento anterior ao cache de assets
- GIVEN um conjunto fixo de dados de assinatura **com registro preenchido**
- AND uma referência produzida pela implementação anterior com esses dados
- WHEN a implementação atual compõe a imagem
- THEN o resultado é idêntico pixel a pixel
- AND a única divergência intencional é a omissão do separador quando o registro
  está vazio (requisito abaixo)

---

### Requirement: Separador omitido quando um dos lados está vazio

Linhas compostas por dois campos opcionais só recebem o separador quando ambos
os lados têm conteúdo. A comparação despreza espaços em branco.

| Linha | Campos | Separador |
|-------|--------|-----------|
| Profissão | profissão + registro | ` - ` |
| Contato | celular + telefone fixo | ` / ` |

Ambas usam o mesmo helper, `juntarCampos`, que difere apenas no separador.

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

#### Scenario: Apenas um telefone informado
- GIVEN celular preenchido e telefone fixo vazio (ou o inverso)
- WHEN a linha de contato é composta
- THEN apenas o telefone informado é desenhado, sem a barra separadora

---

### Requirement: Declaração de dependências

`golang.org/x/image` é dependência direta, por fornecer o tipo `font.Face` usado
nos campos do renderer.

O parse das fontes é feito pela própria `gg.LoadFontFace` (função exportada no
nível do pacote), e **não** por `truetype.Parse` direto — é literalmente o mesmo
código que `dc.LoadFontFace` chama por baixo, o que elimina qualquer risco de
divergência no resultado. Por isso `github.com/golang/freetype` permanece uma
dependência indireta.

#### Scenario: Árvore de build enxuta
- GIVEN o `go.mod` atual
- WHEN `go mod tidy` é executado
- THEN nenhum módulo é adicionado ou removido do `go.sum`

---

## Deprecated

### Requirement: Carregamento de fonte por chamada de desenho (Removido: 2026-08-14)

As chamadas a `dc.LoadFontFace("./Arial.ttf", ...)` intercaladas com o desenho
dos textos deixaram de existir.

**Motivo:** reparsear ~1 MB de arquivos TrueType a cada composição era tolerável
quando a composição ocorria uma vez por clique, mas inviabiliza a renderização
por tecla digitada.

**Cuidado ao reintroduzir:** `dc.LoadFontFace` e `dc.SetFontFace` calculam
`dc.fontHeight` de formas diferentes (`points*72/96` contra as métricas da
fonte). A diferença só chega ao resultado pelo termo `y += ay*h` de
`DrawStringAnchored`, e todas as chamadas atuais usam `ay = 0` — por isso os dois
caminhos são equivalentes hoje. Um `ay` diferente de zero passaria a depender
desse detalhe.
