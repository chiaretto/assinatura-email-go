# Project: assinatura-email-go

Gerador desktop de assinaturas de e-mail em imagem (PNG) para o escritório HFCD.

## Stack

| Camada | Tecnologia |
|--------|-----------|
| Linguagem | Go 1.23 |
| GUI | [Fyne](https://fyne.io) v2.5.1 |
| Composição de imagem | [fogleman/gg](https://github.com/fogleman/gg) v1.3.0 |
| Build/CI | GitHub Actions (`.github/workflows/build.yaml`) |

## Estrutura

```
main.go                  # aplicação inteira: renderização + UI
render_test.go           # equivalência de renderização e benchmarks
ui_test.go               # preview e modo degradado (headless, via fyne/test)
modelo-assinatura.png    # imagem de fundo (1604x287, PNG RGBA 8-bit)
Arial.ttf / ArialBold.ttf# fontes usadas na composição
rodape.jpg               # asset auxiliar
```

Organização do `main.go`, de cima para baixo: constantes de asset e layout,
`signatureData`, `renderer` (`newRenderer` / `render` / `save`), `main` e
`montarPreview`.

## Convenções

- Código em um único pacote `main`; funções auxiliares no mesmo arquivo enquanto
  o escopo permanecer pequeno.
- Textos de UI e mensagens de erro em **português**.
- Comentários explicam o *porquê*, não o *o quê*, e acompanham funções não óbvias.
- Assets são carregados por caminho relativo ao diretório de trabalho
  (`modelo-assinatura.png`, `./Arial.ttf`) e **uma única vez**, no startup, para
  dentro do `renderer`. Nada de I/O no caminho de renderização.
- A renderização é a única fonte de verdade: preview e arquivo salvo saem da
  mesma função, então não podem divergir.
- Erros de renderização são propagados como `error` e exibidos via
  `dialog.ShowError`; falha ao carregar assets degrada a UI em vez de derrubar
  o app.
- O `renderer` não é seguro para uso concorrente (as `font.Face` da `gg` não
  são): renderizar fora da thread de UI exige serializar o acesso.

## Coordenadas do modelo (referência)

Sobre `modelo-assinatura.png` (1604x287), origem no canto superior esquerdo.
Nota: o `main.go` original declarava `const width = 2465` / `height = 439`, mas
eram constantes mortas — nunca usadas e divergentes do modelo real. Foram
removidas no refactor de desempenho.

| Campo | Fonte | Tamanho | X | Y |
|-------|-------|---------|---|---|
| Nome | Arial | 40 | 300 | 162 |
| Profissão + registro | Arial | 30 | 300 | 198 |
| E-mail | Arial | 30 | 300 | 265 |
| Telefones | ArialBold | 25 | 900 | 148 |

Cor do texto: `#4C4C4C`.

As linhas compostas por dois campos opcionais (profissão + registro, celular +
telefone) usam `juntarCampos`, que omite o separador quando um dos lados está
vazio — nada de traço ou barra órfãos na imagem.
