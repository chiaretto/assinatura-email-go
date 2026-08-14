# Delta: Interface do Usuário

**Change ID:** `add-realtime-signature-preview`
**Affects:** `main.go` (`main()`), layout da janela, handlers dos campos

---

## ADDED

### Requirement: Preview visual da assinatura na janela principal

A janela principal exibe, abaixo do formulário, uma representação visual da
assinatura que será gerada, composta a partir de `modelo-assinatura.png` com os
textos dos campos sobrepostos.

#### Scenario: Preview visível ao abrir o aplicativo
- GIVEN que o aplicativo foi iniciado e os assets carregaram com sucesso
- WHEN a janela principal é exibida
- THEN o preview já mostra o modelo de assinatura renderizado com o estado
  inicial dos campos (todos vazios, exceto o telefone fixo com seu valor padrão)
- AND nenhuma interação prévia do usuário é necessária

#### Scenario: Preview mantém a proporção do modelo
- GIVEN um modelo de 1604x287 pixels (proporção ≈5,6:1)
- WHEN o preview é exibido em uma área menor que o tamanho nativo
- THEN a imagem é escalada preservando a proporção (`ImageFillContain`),
  sem distorção nem corte

---

### Requirement: Atualização em tempo real ao digitar

Qualquer alteração no conteúdo de qualquer campo do formulário atualiza o
preview imediatamente, sem clique em botão.

#### Scenario: Digitação no campo Nome
- GIVEN o aplicativo aberto com o preview visível
- WHEN o usuário digita "Fabiano Chiaretto Fernandes" no campo Nome
- THEN o preview passa a exibir esse nome na posição do nome, com a mesma
  fonte, tamanho e cor que o arquivo final terá

#### Scenario: Alteração em qualquer um dos seis campos
- GIVEN o aplicativo aberto
- WHEN o usuário altera Nome, E-mail, Profissão, Registro, Celular **ou**
  Telefone
- THEN o preview reflete a alteração
- AND a linha de contato continua respeitando a regra de `juntarTelefones`
  (sem o separador " / " quando um dos telefones está vazio)
- AND a linha de profissão respeita a regra de `juntarProfissao`
  (sem o separador " - " quando o registro está vazio)

#### Scenario: Registro apagado com o preview aberto
- GIVEN um preview exibindo "Advogado - OAB/MG - 123456"
- WHEN o usuário apaga todo o conteúdo do campo Registro
- THEN o preview passa a exibir apenas "Advogado", sem traço órfão

#### Scenario: Limpar um campo
- GIVEN um campo previamente preenchido
- WHEN o usuário apaga todo o seu conteúdo
- THEN o preview volta a exibir aquela área sem texto

#### Scenario: Digitação contínua permanece fluida
- GIVEN o usuário digitando em ritmo normal
- WHEN caracteres são inseridos em sequência rápida
- THEN a interface não apresenta travamento perceptível
- AND o preview converge para o estado final do texto digitado

---

### Requirement: Modo degradado quando os assets não carregam

A ausência ou corrupção dos arquivos de asset não pode derrubar o aplicativo.

#### Scenario: Imagem de modelo ausente no startup
- GIVEN que `modelo-assinatura.png` não existe no diretório de trabalho
- WHEN o aplicativo é iniciado
- THEN a janela abre normalmente
- AND uma mensagem de erro legível é exibida na área do preview
- AND o botão "Gerar Assinatura de Email" está desabilitado
- AND o processo não sofre `panic` nem encerra

#### Scenario: Arquivo de fonte ausente no startup
- GIVEN que `Arial.ttf` ou `ArialBold.ttf` não existe no diretório de trabalho
- WHEN o aplicativo é iniciado
- THEN o comportamento é o mesmo do cenário anterior, com a mensagem indicando
  qual asset falhou

---

## MODIFIED

### Requirement: Layout e dimensões da janela principal

Antes: um `container.NewVBox` com seis campos e um botão, em uma janela de
400x300.

Agora: um `container.NewBorder` com o formulário no topo e o preview ocupando a
área central, em uma janela dimensionada para acomodar ambos (referência:
760x480).

#### Scenario: Janela redimensionada pelo usuário
- GIVEN a janela principal exibida
- WHEN o usuário aumenta ou reduz o tamanho da janela
- THEN o formulário mantém sua altura natural no topo
- AND o preview ocupa o espaço restante, sempre preservando a proporção

---

### Requirement: Feedback ao gerar o arquivo

Antes: o diálogo de sucesso exibia apenas o nome do arquivo
(`assinatura-<nome>.png`), sem indicar sua localização.

Agora: o diálogo exibe o **caminho absoluto** do arquivo salvo.

#### Scenario: Geração bem-sucedida
- GIVEN todos os campos obrigatórios preenchidos (nome, e-mail, profissão)
- WHEN o usuário clica em "Gerar Assinatura de Email"
- THEN o arquivo é salvo
- AND o diálogo de sucesso exibe o caminho absoluto do arquivo
- AND o conteúdo salvo é idêntico ao que estava sendo exibido no preview

#### Scenario: Campos obrigatórios em branco
- GIVEN que nome, e-mail ou profissão está vazio
- WHEN o usuário clica em "Gerar Assinatura de Email"
- THEN o diálogo de erro "Todos os campos devem ser preenchidos" é exibido
  (comportamento inalterado)
- AND o preview permanece visível, refletindo o estado parcial dos campos —
  a validação bloqueia apenas a gravação em disco, nunca o preview

---

## REMOVED

(Nenhum requisito removido.)
