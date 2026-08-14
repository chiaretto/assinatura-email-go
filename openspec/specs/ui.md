# Interface do Usuário

Fonte de verdade sobre a janela principal, o preview e o fluxo de geração.

**Última atualização:** 2026-08-14 (change `add-realtime-signature-preview`)

---

## Requirements

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
- AND a altura mínima é derivada da proporção real do modelo, não de um valor
  fixo

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
- AND as linhas compostas respeitam a regra de omissão de separador definida em
  [Renderização da Assinatura](signature-rendering.md)

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

  *A atualização é síncrona no `OnChanged`, sem debounce: a renderização custa
  ~2,3 ms, muito abaixo do intervalo entre teclas. Isso também evita tocar no
  canvas fora da thread de UI — o Fyne 2.5.1 não expõe `fyne.Do`.*

#### Scenario: Texto que excede a largura do modelo
- GIVEN um nome ou linha de profissão longo demais para o espaço disponível
- WHEN o preview é atualizado
- THEN o estouro fica visível para o usuário antes de gerar o arquivo
- AND o texto não é truncado, quebrado nem reduzido automaticamente

---

### Requirement: Modo degradado quando os assets não carregam

A ausência ou corrupção dos arquivos de asset não pode derrubar o aplicativo.

#### Scenario: Imagem de modelo ausente no startup
- GIVEN que `modelo-assinatura.png` não existe no diretório de trabalho
- WHEN o aplicativo é iniciado
- THEN a janela abre normalmente
- AND uma mensagem de erro legível é exibida na área do preview, indicando os
  arquivos esperados
- AND o botão "Gerar Assinatura de Email" está desabilitado
- AND o processo não sofre `panic` nem encerra

#### Scenario: Arquivo de fonte ausente no startup
- GIVEN que `Arial.ttf` ou `ArialBold.ttf` não existe no diretório de trabalho
- WHEN o aplicativo é iniciado
- THEN o comportamento é o mesmo do cenário anterior, com a mensagem indicando
  qual asset falhou

---

### Requirement: Layout e dimensões da janela principal

Um `container.NewBorder` com o formulário no topo e o preview ocupando a área
central.

#### Scenario: Janela redimensionada pelo usuário
- GIVEN a janela principal exibida
- WHEN o usuário aumenta ou reduz o tamanho da janela
- THEN o formulário mantém sua altura natural no topo
- AND o preview ocupa o espaço restante, sempre preservando a proporção

---

### Requirement: Campos obrigatórios e opcionais

Nome, e-mail e profissão são obrigatórios para **gravar** o arquivo. Registro,
celular e telefone fixo são opcionais. Nenhum campo é obrigatório para o
preview.

#### Scenario: Campos obrigatórios em branco
- GIVEN que nome, e-mail ou profissão está vazio
- WHEN o usuário clica em "Gerar Assinatura de Email"
- THEN o diálogo de erro "Todos os campos devem ser preenchidos" é exibido
- AND o preview permanece visível, refletindo o estado parcial dos campos —
  a validação bloqueia apenas a gravação em disco, nunca o preview

---

### Requirement: Feedback ao gerar o arquivo

O diálogo de sucesso exibe o **caminho absoluto** do arquivo salvo.

#### Scenario: Geração bem-sucedida
- GIVEN todos os campos obrigatórios preenchidos (nome, e-mail, profissão)
- WHEN o usuário clica em "Gerar Assinatura de Email"
- THEN o arquivo é salvo como `assinatura-<nome>.png` no diretório de trabalho
- AND o diálogo de sucesso exibe o caminho absoluto do arquivo
- AND o conteúdo salvo é idêntico ao que estava sendo exibido no preview

#### Scenario: Caminho absoluto não pode ser resolvido
- GIVEN que o arquivo foi salvo com sucesso
- WHEN `filepath.Abs` falha
- THEN o diálogo exibe o caminho relativo
- AND nenhum erro é reportado — a gravação de fato aconteceu
