# assinatura-email-go

Aplicativo desktop que gera assinaturas de e-mail em imagem (PNG) para o
escritório HFCD. Preencha os campos e veja o resultado se formar em tempo real
antes de salvar.

## Uso

1. Preencha nome, e-mail, profissão, registro e telefones.
2. O preview na janela mostra exatamente como a imagem vai ficar — o mesmo
   código que desenha o preview é o que grava o arquivo.
3. Clique em **Gerar Assinatura de Email**. O arquivo é salvo como
   `assinatura-<nome>.png` no diretório de trabalho, e o caminho completo
   aparece na confirmação.

Nome, e-mail e profissão são obrigatórios; registro e telefones são opcionais.
Os separadores só aparecem quando há texto dos dois lados: sem registro, a linha
mostra apenas a profissão (sem ` - `); com apenas um telefone, o ` / ` é
omitido.

## Assets

O aplicativo carrega estes arquivos **do diretório de trabalho**, então eles
precisam estar junto do executável:

| Arquivo | Uso |
|---------|-----|
| `modelo-assinatura.png` | imagem de fundo (1604x287) |
| `Arial.ttf` | nome, profissão/registro e e-mail |
| `ArialBold.ttf` | linha de telefones |

Se algum deles faltar, a janela ainda abre: o preview é substituído por uma
mensagem indicando o que não foi encontrado, e a geração fica desabilitada.

## Desenvolvimento

```bash
go build            # compila o binário
go test ./...       # testes de renderização e de UI (headless)
go vet ./...
go test -bench=Render -benchmem   # compara o render atual com o antigo
```

Os testes cobrem a equivalência pixel a pixel entre o preview e o arquivo
salvo, além de garantir que o refactor de desempenho não alterou a imagem
produzida.

O processo de mudanças segue o fluxo OpenSpec — veja `openspec/project.md`.
